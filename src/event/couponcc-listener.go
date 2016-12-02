/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gogap/logrus"
	"github.com/gogap/logrus/hooks/file"
	"github.com/hyperledger/fabric/events/consumer"
	pb "github.com/hyperledger/fabric/protos"
	"os"
)

type adapter struct {
	notfy              chan *pb.Event_Block
	rejected           chan *pb.Event_Rejection
	cEvent             chan *pb.Event_ChaincodeEvent
	listenToRejections bool
	chaincodeID        string
}

type cc_message struct {
	Type string   `json:"type,omitempty"`
	Txs  []string `json:"txs,omitempty"`
	Msg  string   `json:"msg,omitempty"`
}

var logger = logrus.New()

//GetInterestedEvents implements consumer.EventAdapter interface for registering interested events
func (a *adapter) GetInterestedEvents() ([]*pb.Interest, error) {
	if a.chaincodeID != "" {
		return []*pb.Interest{
			{EventType: pb.EventType_BLOCK},
			{EventType: pb.EventType_REJECTION},
			{EventType: pb.EventType_CHAINCODE,
				RegInfo: &pb.Interest_ChaincodeRegInfo{
					ChaincodeRegInfo: &pb.ChaincodeReg{
						ChaincodeID: a.chaincodeID,
						EventName:   ""}}}}, nil
	}
	return []*pb.Interest{{EventType: pb.EventType_BLOCK}, {EventType: pb.EventType_REJECTION}}, nil
}

//Recv implements consumer.EventAdapter interface for receiving events
func (a *adapter) Recv(msg *pb.Event) (bool, error) {
	if o, e := msg.Event.(*pb.Event_Block); e {
		a.notfy <- o
		return true, nil
	}
	if o, e := msg.Event.(*pb.Event_Rejection); e {
		if a.listenToRejections {
			a.rejected <- o
		}
		return true, nil
	}
	if o, e := msg.Event.(*pb.Event_ChaincodeEvent); e {
		a.cEvent <- o
		return true, nil
	}
	return false, fmt.Errorf("Receive unkown type event: %v", msg)
}

//Disconnected implements consumer.EventAdapter interface for disconnecting
func (a *adapter) Disconnected(err error) {
	fmt.Printf("Disconnected...exiting\n")
	os.Exit(1)
}

func createEventClient(eventAddress string, listenToRejections bool, cid string) (*adapter, error) {
	var obcEHClient *consumer.EventsClient

	done := make(chan *pb.Event_Block)
	reject := make(chan *pb.Event_Rejection)
	adapter := &adapter{notfy: done, rejected: reject, listenToRejections: listenToRejections, chaincodeID: cid, cEvent: make(chan *pb.Event_ChaincodeEvent)}
	obcEHClient, _ = consumer.NewEventsClient(eventAddress, 5, adapter)
	if err := obcEHClient.Start(); err != nil {
		fmt.Printf("could not start chat %s\n", err)
		obcEHClient.Stop()
		return nil, err
	}

	return adapter, nil
}

func initLog(logFile, logLevel string) {
	logger.Formatter = new(logrus.JSONFormatter)
	//logger.Formatter = new(logrus.TextFormatter) // default
	switch logLevel {
	case "DEBUG":
		logger.Level = logrus.DebugLevel
	case "INFO":
		logger.Level = logrus.InfoLevel
	case "WARN":
		logger.Level = logrus.WarnLevel
	case "ERROR":
		logger.Level = logrus.ErrorLevel
	case "FATAL":
		logger.Level = logrus.FatalLevel
	case "PANIC":
		logger.Level = logrus.PanicLevel
	default:
		logger.Level = logrus.InfoLevel
	}
	logger.Hooks.Add(file.NewHook(logFile))
}
func main() {
	var eventAddress string
	var listenToRejections bool
	var chaincodeID string
	var amqpFile string
	var logFile string
	var logLevel string
	pubChan := make(chan string)
	flag.StringVar(&eventAddress, "events-address", "0.0.0.0:7053", "address of events server")
	flag.BoolVar(&listenToRejections, "listen-to-rejections", false, "whether to listen to rejection events")
	flag.StringVar(&chaincodeID, "events-from-chaincode", "", "listen to events from given chaincode")
	flag.StringVar(&amqpFile, "amqp-file", "amqp.yaml", "amqp config file, yaml style")
	flag.StringVar(&logFile, "log-file", "logs/ccevent.log", "log file")
	flag.StringVar(&logLevel, "log-level", "DEBUG", "log level:  DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.Parse()
	initLog(logFile, logLevel)
	p, err := NewPublisher(amqpFile, pubChan, logger)
	if err != nil || p == nil {
		logger.Errorf("NewPublisher: %v", err)
		return
	} else {
		logger.Info("publisher created..")

	}
	a, err := createEventClient(eventAddress, listenToRejections, chaincodeID)
	if err != nil {
		logger.Errorf("Error creating event client: [%v]", err)
		return
	} else {
		logger.Info("chatting with cc.....")

	}
	p.SpinSend()
	for {
		select {
		case b := <-a.notfy:
			//block created event
			txCount := len(b.Block.Transactions)
			txs := make([]string, txCount, txCount)
			i := 0
			for _, r := range b.Block.Transactions {
				logger.WithFields(logrus.Fields{
					"event": "BLOCK_EVENT",
					"type":  "EACH_TX",
					"txID":  r.Txid,
				}).Infof("Transaction: [%v]", r)
				txs[i] = r.Txid
				i++
			}
			cc_msg := &cc_message{Type: "BLOCK_EVENT", Txs: txs}
			bytes, err := json.Marshal(cc_msg)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"event": "BLOCK_EVENT",
					"type":  "PUB_TXS",
				}).Warnf("json marshal err: [%v]", err)
			}
			logger.Debugf("publish json msg: [%s]", string(bytes))
			pubChan <- string(bytes)
		case r := <-a.rejected:
			logger.WithFields(logrus.Fields{
				"event": "REJECTION_EVETN",
				"txID":  r.Rejection.Tx.Txid,
			}).Warnf("tx rejected: [%s]", r.Rejection.ErrorMsg)
			cc_msg := &cc_message{Type: "REJECTION_EVENT", Txs: []string{r.Rejection.Tx.Txid}, Msg: r.Rejection.ErrorMsg}
			bytes, err := json.Marshal(cc_msg)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"event": "REJECTION_EVENT",
					"type":  "PUB_TX",
				}).Warnf("json marshal err: [%v]", err)
			}
			logger.Debugf("json-msg:%s", string(bytes))
			pubChan <- string(bytes)
		case ce := <-a.cEvent:
			logger.WithFields(logrus.Fields{
				"event": "CHAINCODE_EVENT",
			}).Infof("chaincode event: [%v]", ce.ChaincodeEvent)
			cc_msg := &cc_message{Type: "CHAINCODE_EVENT", Txs: []string{ce.ChaincodeEvent.TxID}}
			bytes, err := json.Marshal(cc_msg)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"event": "CHAINCODE_EVENT",
					"type":  "PUB_TX",
				}).Warnf("json marshal err: [%v]", err)
			}
			pubChan <- string(bytes)
		}
	}
}
