package main

import (
	"fmt"
	log "github.com/gogap/logrus"
	"github.com/gogap/logrus/hooks/file"
)

func main() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{})
	log.AddHook(file.NewHook("logs/test.log"))
	fmt.Print("hello\n")
	log.WithField("biz0", "member").Debugf("member:[%s] not login", "huajie")
	fmt.Print("world\n")
	log.WithField("biz", "member").Errorf("member:[%s] not login", "huajie")

}
