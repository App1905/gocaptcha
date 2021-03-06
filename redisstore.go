package gocaptcha

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gopkg.in/redis.v2"
)

func init() {
	RegisterStore(storeName, CreateCaptchaRedisStore)
}

const (
	captchaKeyFormat = "captcha:text:%s;rand:%s;time:%x;"
	storeName        = "redis"
)

type CaptchaRedisStore struct {
	lifeTime time.Duration
	stg      *redis.Client
}

func CreateCaptchaRedisStore(config *StoreConfig) (StoreInterface, error) {
	lifeTime := config.LifeTime
	if config.Servers == nil || len(config.Servers) == 0 {
		return nil, fmt.Errorf("servers must not be empty")
	}

	fullAddr := strings.TrimPrefix(config.Servers[0], "redis://")
	pieces := strings.SplitN(fullAddr, "/", 2)

	db := 0
	addr := pieces[0]
	if len(pieces) == 2 {
		db, _ = strconv.Atoi(pieces[1])
	}

	opt := redis.Options{}
	opt.Addr = addr
	opt.DB = int64(db)
	opt.PoolSize = 0
	stg := redis.NewTCPClient(&opt)

	return &CaptchaRedisStore{lifeTime, stg}, nil
}

func (this *CaptchaRedisStore) Get(key string) *CaptchaInfo {
	s, err := this.stg.Get(key).Result()
	if err != nil {
		log.Printf("get key in redis error:%s", err)
		return nil
	}

	captcha := CaptchaInfo{}
	this.decodeCaptachInfo([]byte(s), &captcha)
	return &captcha
}

func (this *CaptchaRedisStore) Add(captcha *CaptchaInfo) string {
	key := fmt.Sprintf(captchaKeyFormat, captcha.Text, randStr(20), captcha.CreateTime.Unix())
	key = hex.EncodeToString(md5.New().Sum([]byte(key)))
	key = key[:32]

	val, err := this.encodeCaptchaInfo(captcha)
	if err == nil {
		if seterr := this.stg.SetEx(key, this.lifeTime, string(val)); seterr.Err() != nil {
			log.Printf("add key in redis error:%s", seterr)
		}
	}
	return key
}

func (this *CaptchaRedisStore) Update(key string, captcha *CaptchaInfo) bool {
	val, err := this.encodeCaptchaInfo(captcha)
	if err == nil {
		if seterr := this.stg.Set(key, string(val)); seterr.Err() != nil {
			log.Printf("set key in redis error:%s", seterr)
			return false
		} else {
			return true
		}
	} else {
		return false
	}

}

func (this *CaptchaRedisStore) Del(key string) {
	this.stg.Del(key)
}

func (this *CaptchaRedisStore) Destroy() {

}

func (this *CaptchaRedisStore) OnConstruct() {

}

func (this *CaptchaRedisStore) OnDestruct() {

}

func (this *CaptchaRedisStore) encodeCaptchaInfo(captcha *CaptchaInfo) ([]byte, error) {
	return json.Marshal(captcha)
}

func (this *CaptchaRedisStore) decodeCaptachInfo(b []byte, ret *CaptchaInfo) {
	json.Unmarshal(b, ret)
	return
}
