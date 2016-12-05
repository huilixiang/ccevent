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
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/events/consumer"
	pb "github.com/hyperledger/fabric/protos"
	"os"
)

type adapter struct {
	notify             chan *pb.Event_Block
	rejected           chan *pb.Event_Rejection
	cEvent             chan *pb.Event_ChaincodeEvent
	listenToRejections bool
	chaincodeID        string
}

type cc_message struct {
	Type    string `json:"type,omitempty"`
	Txid    string `json:"txid,omitempty"`
	Payload string `json:"payload,omitempty"`
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
		a.notify <- o
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
	adapter := &adapter{notify: done, rejected: reject, listenToRejections: listenToRejections, chaincodeID: cid, cEvent: make(chan *pb.Event_ChaincodeEvent)}
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

func pubEvent(eventType string, chanType string, tx *pb.Transaction, pubChan chan string) error {
	logger.WithFields(logrus.Fields{
		"event": eventType,
		"type":  chanType,
		"txID":  tx.Txid,
	}).Infof("Transaction: [%v]", tx)
	txJson, err := tx2Json(tx)
	if err != nil {
		logger.Warnf("Marshal tx: [%s], error: [%v]", tx.Txid, err)
		return nil
	}
	return pubTxMsg("BLOCK_EVENT", tx.Txid, txJson, pubChan)

}

func tx2Json(tx *pb.Transaction) (string, error) {
	ccis := &pb.ChaincodeInvocationSpec{}
	err := proto.Unmarshal(tx.Payload, ccis)
	if err != nil {
		logger.Warnf("Unmarshal tx: [%s], error: [%v]", tx.Txid, err)
		return "", err
	}
	txJson, err := proto.Marshal(ccis)
	if err != nil {
		logger.Warnf("Marshal tx: [%s], error: [%v]", tx.Txid, err)
		return "", err
	}
	return string(txJson), nil
}

func pubTxMsg(eventType string, txid string, payload string, pubChan chan string) error {
	cc_msg := &cc_message{Type: eventType, Txid: txid, Payload: payload}
	bytes, err := json.Marshal(cc_msg)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"event": eventType,
			"type":  "PUB_TX",
		}).Warnf("json marshal err: [%v]", err)
		return err
	}
	logger.Debugf("publish json msg: [%s]", string(bytes))
	pubChan <- string(bytes)
	return nil
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
		case b := <-a.notify:
			//block created event
			for _, tx := range b.Block.Transactions {
				pubEvent("BLOCK_EVENT", "notify", tx, pubChan)
			}
		case r := <-a.rejected:
			pubEvent("REJECTION_EVETN", "rejected", r.Rejection.Tx, pubChan)
		case ce := <-a.cEvent:
			logger.WithFields(logrus.Fields{
				"event": "CHAINCODE_EVENT",
			}).Infof("chaincode event: [%v]", ce.ChaincodeEvent.String())
			cc_msg := &cc_message{Type: "CHAINCODE_EVENT", Txid: ce.ChaincodeEvent.TxID, Payload: ce.ChaincodeEvent.String()}
			bytes, err := json.Marshal(cc_msg)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"event": "CHAINCODE_EVENT",
					"type":  "cEvent",
				}).Warnf("json marshal err: [%v]", err)
			}
			pubChan <- string(bytes)
		}
	}
}
