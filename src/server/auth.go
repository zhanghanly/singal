package media_center

type AuthParam struct {
	AppId     string
	Nonce     string
	Timestamp string
	Signature string
}

func DoApiAuth(param *AuthParam) (res bool, err error) {
	//1.检查是否在白名单
	for _, v := range gConfig.AuthWhiteList {
		if param.Signature == v.Signature {
			return true, nil
		}
	}
	//2. 如果不在白名单，严格按照鉴权流程走
	return false, nil
}
