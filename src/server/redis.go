package singal

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	// "strconv"
	"strings"
	"sync"
	"time"
)

var gRedisClient *redis.Client

type RedisTaskMgr struct {
}

var gRedisTaskMgr *RedisTaskMgr

func InitRedisClient() (err error) {
	logger.Infof("redisinit host:%s passwd:%s", gConfig.Redis.Host, gConfig.Redis.Password)
	gRedisClient = redis.NewClient(&redis.Options{
		Addr:     gConfig.Redis.Host,
		Password: gConfig.Redis.Password,
		DB:       0,
	})
	gRedisTaskMgr = &RedisTaskMgr{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = gRedisClient.Ping(ctx).Result()
	return err
}

// redis key --------------------------
type RedisKeyValueGeneratorFunc func(args *RedisScriptArguments) string

type RedisKey struct {
	id           string
	staticValue  string
	dynamicValue RedisKeyValueGeneratorFunc
}

func NewStaticKey(id string, value string) *RedisKey {
	return &RedisKey{
		id:           id,
		staticValue:  value,
		dynamicValue: nil,
	}
}

func NewDynamicKey(id string, generator RedisKeyValueGeneratorFunc) *RedisKey {
	return &RedisKey{
		id:           id,
		dynamicValue: generator,
	}
}

func (this *RedisKey) Key() string {
	return this.id
}

func (this *RedisKey) Value(args *RedisScriptArguments) string {
	if this.dynamicValue == nil {
		return this.staticValue
	} else {
		return this.dynamicValue(args)
	}
}

//redis key --------------------------

type RedisScript struct {
	scriptText string
	args       []string
	keys       []string
}

func NewRedisScript(keys []string, args []string, scriptText string) *RedisScript {
	return &RedisScript{
		scriptText: scriptText,
		args:       args,
		keys:       keys,
	}
}

func getScriptsUniqueArgNames(scripts []*RedisScript) []string {
	uniqueArgs := make(map[string]bool, 0)
	uniqueArgsSlice := []string{}

	for _, script := range scripts {
		for _, key := range script.args {
			if !uniqueArgs[key] {
				uniqueArgsSlice = append(uniqueArgsSlice, key)
				uniqueArgs[key] = true
			}
		}
	}

	return uniqueArgsSlice
}

func joinRedisScripts(scripts []*RedisScript, keys []*RedisKey, args []string) *RedisScript {
	result := &RedisScript{}

	var functionCalls []string

	for scriptIndex, script := range scripts {
		compiledArgs := ""
		for argIndex, arg := range args {
			result.args = append(result.args, arg)
			compiledArgs = compiledArgs + fmt.Sprintf("local %s = ARGV[%d];\n", arg, argIndex+1)
		}

		compiledKeys := ""
		for keyIndex, key := range keys {
			result.keys = append(result.keys, key.Key())
			compiledKeys = compiledKeys + fmt.Sprintf("local %s = KEYS[%d];\n", key.Key(), keyIndex+1)
		}

		functionName := fmt.Sprintf("____joinedRedisScripts_%d____", scriptIndex)

		envelopedScriptText := fmt.Sprintf("local function %s()\n%s\n%s\n%s\nend", functionName, compiledKeys, compiledArgs, script.scriptText)

		functionCalls = append(functionCalls, fmt.Sprintf("%s()", functionName))

		if len(result.scriptText) > 0 {
			result.scriptText = result.scriptText + "\n" + envelopedScriptText
		} else {
			result.scriptText = envelopedScriptText
		}
	}

	result.scriptText = result.scriptText + "\n" + "return {" + strings.Join(functionCalls, ", ") + "}\n"

	return result
}

func (this *RedisScript) String() string {
	return this.scriptText
}

func (this *RedisScript) Keys() []string {
	return this.keys
}

func (this *RedisScript) Args() []string {
	return this.args
}

type RedisScriptArguments map[string]interface{}
type CompiledRedisScript struct {
	script      RedisScript
	scriptText  string
	keys        []*RedisKey
	args        []string
	redisScript *redis.Script
	mx          sync.RWMutex
}

func getUniqueKeys(keys []*RedisKey) []*RedisKey {
	uniqueKeys := make(map[string]bool, 0)
	uniqueKeysSlice := []*RedisKey{}

	for _, key := range keys {
		if !uniqueKeys[key.Key()] {
			uniqueKeysSlice = append(uniqueKeysSlice, key)
			uniqueKeys[key.Key()] = true
		} else {
			panic("Duplicate key: " + key.Key())
		}
	}

	return uniqueKeysSlice
}

func getUniqueUsedKeys(scripts []*RedisScript, keys []*RedisKey) []*RedisKey {
	usedKeys := make(map[string]bool, 0)

	for _, script := range scripts {
		for _, key := range script.keys {
			usedKeys[key] = true
		}
	}

	uniqueKeys := make(map[string]bool, 0)
	keysSlice := []*RedisKey{}

	for _, key := range keys {
		if usedKeys[key.Key()] {
			if !uniqueKeys[key.Key()] {
				keysSlice = append(keysSlice, key)

				uniqueKeys[key.Key()] = true
			}
		}
	}

	return keysSlice
}

func CompileRedisScripts(scripts []*RedisScript, keys []*RedisKey) (*CompiledRedisScript, error) {
	suppliedKeys := make(map[string]*RedisKey)

	for _, key := range keys {
		suppliedKeys[key.Key()] = key
	}

	for _, key := range keys {
		if suppliedKeys[key.Key()] == nil {
			return nil, fmt.Errorf("Missing required LUA script key: %v", key)
		}
	}

	uniqueKeys := getUniqueUsedKeys(scripts, keys)
	uniqueArgs := getScriptsUniqueArgNames(scripts)

	script := joinRedisScripts(scripts, uniqueKeys, uniqueArgs)

	result := &CompiledRedisScript{
		script:     *script,
		scriptText: script.scriptText,
		keys:       uniqueKeys,
		args:       uniqueArgs,
	}

	return result, nil
}

func (this *CompiledRedisScript) String() string {
	return this.scriptText
}

func (this *CompiledRedisScript) Keys(args *RedisScriptArguments) []string {
	var result []string = []string{}

	for _, key := range this.keys {
		result = append(result, key.Value(args))
	}

	return result
}

func (this *CompiledRedisScript) Args(args *RedisScriptArguments) ([]interface{}, error) {
	var result []interface{} = []interface{}{}

	for _, arg := range this.script.args {
		value, ok := (*args)[arg]

		if !ok {
			return nil, fmt.Errorf("Missing required Redis LUA script argument: %v", arg)
		}

		result = append(result, value)
	}

	return result, nil
}

func (this *CompiledRedisScript) Run(ctx context.Context, client *redis.Client, args *RedisScriptArguments) *redis.Cmd {
	if this.redisScript == nil {
		this.mx.Lock()
		this.redisScript = redis.NewScript(this.scriptText)
		this.mx.Unlock()
	}

	if orderedArgsValues, err := this.Args(args); err == nil {
		result := this.redisScript.Run(ctx, client, this.Keys(args), orderedArgsValues)

		if result.Err() != nil && ctx.Err() == nil {
			panic(fmt.Sprintf("Script run error: %v\nKeys: %v\nArgs: %v\nScript: %v\n\n", result.Err(), this.Keys(args), orderedArgsValues, this.scriptText))
		}

		return result
	} else {
		panic(err)
	}
}

func (this *CompiledRedisScript) RunDebug(ctx context.Context, client *redis.Client, args *RedisScriptArguments) *redis.Cmd {
	if this.redisScript == nil {
		this.mx.Lock()
		this.redisScript = redis.NewScript(this.scriptText)
		this.mx.Unlock()
	}

	if orderedArgsValues, err := this.Args(args); err == nil {
		fmt.Printf("RunDebug:\n\tKeys: %v\n\tArgs: %v\nScript: %v\n\n\n", this.Keys(args), orderedArgsValues, this.scriptText)
		result := this.redisScript.Run(ctx, client, this.Keys(args), orderedArgsValues)

		if result.Err() != nil && ctx.Err() == nil {
			panic(fmt.Sprintf("Script run error: %v\nKeys: %v\nArgs: %v\nScript: %v\n\n", result.Err(), this.Keys(args), orderedArgsValues, this.scriptText))
		}

		return result
	} else {
		panic(err)
	}
}

//下面为业务逻辑--------------------------------------

// value {mcId}
func getMediaCenterSetKey() string {
	return "mediacenter:set"
}

//	value {
//		"ip":Ip  // mediacenter实例的ip
//		"port":Port // mediacenter实例的Port
//		"ts": iHeartbeat //mediacenter心跳的时间戳
//		"ipStr": iIpStr //tm实例的字符串ip
//	}
func getMediaCenterStatusKey(mcId string) string {
	return "mediacenterstatus:" + mcId
}

// value {taskname(由deviceID+channelID+tag)}
func getMediaCenterTaskSetKey() string {
	return "mediacenter:taskSet"
}

//	value {
//		status: 0表示有此任务但是流媒体服务还未收到流, 1 表示此任务已经在工作中
//		deviceId:
//		areaType
//		channelId
//		transMode
//		tag
//		timeout
//		packetLossRate
//		mcId 表示此任务被哪个mc来进行管理
//		zlmIp 此任务被哪个媒体服务运行
//		zlmport 此任务的端口号
//	}
func getMediaCenterTaskKey(taskname string) string {
	return "mediacenter:task:" + taskname
}

// 等待任务的集合, 表示mc受理了此任务，但是流媒体还没接收到流
func getAwaitTaskSetKey() string {
	return "mediacenter:awaitTaskSet"
}

//	value {
//		taskname 任务名
//	}
func getMediaCenterTasksKey(mcId string) string {
	return "mediacenter:tasks:" + mcId
}

// value {taskname: 任务名}
func getZLMTasksKey(zlmId string) string {
	return "mediacenter:zlmtasks:" + zlmId
}

//	value {
//		ip:
//		real_play_port:
//	 play_back_port:
//		heartbeat:
//		cpu:
//		bandwidthUsage:
//		allBandwidth:
//	}
func getMediaCenterZLMStatusKey(zlmId string) string {
	return "mediacenter:zlmstatus:" + zlmId
}

func getMediaCenterZLMPushStreamNumsKey(zlmId string) string {
	return "mediacenter:zlmPushStreamNums:" + zlmId
}

func getMediaCenterZLMBandwidthUsageKey(zlmId string) string {
	return "mediacenter:zlmBandwidthUsage:" + zlmId
}

func getMediaCenterAllocateNodeKey(deviceId string) string {
	return "mediacenter:zlmAllocateNode:" + deviceId
}

func LoadRedisScript() {
	scriptAddTask := `redis.call('SET', key1, arg1); redis.call('SET', key2, arg2); return redis.call('GET',key2);`
	script1 := NewRedisScript([]string{"key1", "key2"}, []string{"arg1", "arg2"}, scriptAddTask)

	compiled, err := CompileRedisScripts(
		[]*RedisScript{script1},
		[]*RedisKey{
			NewStaticKey("key1", "keyName1"),
			NewStaticKey("key2", "keyName2"),
		},
	)

	if err != nil {
		logger.Error(err)
		panic(err)
	}

	scriptArgs := make(RedisScriptArguments, 0)
	scriptArgs["arg1"] = "arg1_expected_value"
	scriptArgs["arg2"] = "arg2_expected_value"
	result, err := compiled.Run(context.TODO(), gRedisClient, &scriptArgs).StringSlice()
	if err != nil {
		logger.Error(err)
		panic(err)
	}
	logger.Infof("redis result %v", result)
}
