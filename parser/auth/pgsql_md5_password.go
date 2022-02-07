package auth

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/pingcap/errors"
)

// EncodePasswordByMD5 PostgreSQL encode password by md5
func EncodePasswordByMD5(user string, pwd string) string {
	if pwd == "" {
		return pwd
	}

	x := md5.Sum([]byte(pwd + user))
	return "md5" + hex.EncodeToString(x[:])
}

// DecodePasswordByMD5 remove prefix "md5"
func DecodePasswordByMD5(pwd string) (string, error) {
	if len(pwd) <= 3 || pwd[0:3] != "md5" {
		return "", errors.New("decode password string failed")
	}

	return pwd[3:], nil
}

// CheckPassword
//	SERVER:  salt = create_random_salt()
//			 send(salt)
//	CLIENT:  recv(salt)
//			 md5_stage1 = md5Sum(password + user)
//			 md5_stage2 = md5Sum(md5_stage1 + salt)
//			 reply = "md5" + md5_stage2
//	SERVER:  recv(reply)
//			 server_md5_stage1 = md5Sum(password + user)
//			 server_md5_stage2 = md5Sum(server_md5_stage1 + salt)
//			 check(reply == "md5" + server_md5_stage2))
// The password saved by the server is "md5" + server_md5_stage1,
// so you can get server_md5_stage1 directly
func CheckPassword(salt [4]byte, pwd, authentication string) bool {
	y := md5.Sum([]byte(pwd + string(salt[:])))
	pd := "md5" + hex.EncodeToString(y[:])
	return pd == authentication
}
