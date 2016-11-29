package main

//package cclistener

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

type amqpCfg struct {
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	Vhost      string `yaml:"vhost"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	RoutingKey string `yaml:"routingkey"`
	Queue      string `yaml:"queue"`
	Exchange   string `yaml:"exchange"`
	Timeout    int    `yaml:"timeout"`
}

type publisher struct {
	cfg     *amqpCfg
	conn    *amqp.Connection
	channel *amqp.Channel
	done    chan error
	pin     <-chan string
}

func NewPublisher(cfgFile string, pin <-chan string) (*publisher, error) {
	cfg, err := loadCfg(cfgFile)
	if err != nil {
		log.Printf("loadCfg err:%v", err.Error())
		return nil, err
	}
	p := &publisher{
		cfg:  cfg,
		done: make(chan error),
		pin:  pin,
	}
	err = p.connect()
	if err != nil {
		log.Fatalf("connect %v", err)
		return nil, err
	}
	err = p.AnnounceQueue()
	if err != nil {
		return nil, fmt.Errorf("AnnounceQueue :%s", err)
	}
	return p, nil

}

func (p *publisher) SpinSend() {
	go func() {
		for {
			select {
			case msg := <-p.pin:
				err := p.publish(msg)
				if err != nil {
					log.Fatalf("publish :%v", err)
				} else {
					log.Printf("publish %s success", msg)
				}
			case err := <-p.done:
				log.Fatalf("connection :%v", err)
				err = p.reconnect()
				if err != nil {
					log.Fatalf("reconnect %v", err)
				}
			} // end select

		} // end for

	}() // end func

}

func (p *publisher) reconnect() error {
	time.Sleep(30 * time.Second)
	if err := p.connect(); err != nil {
		log.Printf("Could not connect in reconnect call: %v", err.Error())
		return fmt.Errorf("reconnect:%s", err)
	}
	err := p.AnnounceQueue()
	if err != nil {
		return fmt.Errorf("AnnounceQueue :%s", err)
	}
	return nil
}

func (p *publisher) connect() error {
	var err error
	amqpUrl := buildMqUrl(p.cfg)
	log.Printf("amqp url:%s", amqpUrl)
	p.conn, err = amqp.Dial(amqpUrl)
	if err != nil {
		log.Fatalf("amqp.Dial err:%s", err)
		return err
	}
	go func() {
		log.Printf("closing:%s", <-p.conn.NotifyClose(make(chan *amqp.Error)))
		p.done <- errors.New("channel closed.")
	}()
	log.Printf("got connection, getting channel")
	p.channel, err = p.conn.Channel()
	if err != nil {
		return fmt.Errorf("channel:%s", err)
	}
	log.Printf("got channel, declaring exchange:%s", p.cfg.Exchange)
	if err = p.channel.ExchangeDeclare(
		p.cfg.Exchange, // exchange name
		"direct",       // exchange type
		true,           // durable
		false,          // autoDelete
		false,          // internal
		false,          // noWait
		nil,            // args
	); err != nil {
		return fmt.Errorf("Exchange declare:%s", err)
	}

	return nil
}

func (p *publisher) AnnounceQueue() error {
	_, err := p.channel.QueueDeclare(
		p.cfg.Queue, // name
		true,        // durable
		false,       // autodelete
		false,       // exclusive
		false,       // nowait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("Queue declare:%s", err)
	}
	log.Printf("declared queue:%s, binding to Exchange :%s", p.cfg.Queue, p.cfg.Exchange)
	if err = p.channel.QueueBind(
		p.cfg.Queue,
		p.cfg.RoutingKey,
		p.cfg.Exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("Queue bind:%s", err)
	}
	return nil

}

func (p *publisher) publish(pmsg string) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Body:         []byte("Go Go AMQP!"),
	}
	err := p.channel.Publish(p.cfg.Exchange, p.cfg.RoutingKey, false, false, msg)
	if err != nil {
		log.Fatalf("publish :%v", err)
		return err
	}
	return nil
}

func buildMqUrl(cfg *amqpCfg) string {
	return "amqp://" + cfg.Username + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Vhost
}

func loadCfg(cfgFilePath string) (*amqpCfg, error) {
	yamlFile, err := ioutil.ReadFile(cfgFilePath)
	if err != nil {
		return nil, err
	}
	log.Printf("yamlFile:%v", yamlFile)
	var cfg amqpCfg
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, err
	}
	log.Printf("cfg:%v", cfg)
	return &cfg, nil
}

/**
func main() {
	pubChan := make(chan string)
	exitChan := make(chan string)
	p, err := NewPublisher("./amqp.yaml", pubChan)
	if err != nil {
		fmt.Errorf("NewPublisher %v", err)
		return
	}
	p.SpinSend()
	pubChan <- "hello"
	<-exitChan

}
*/
