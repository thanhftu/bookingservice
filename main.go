package main

import (
	"flag"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/streadway/amqp"
	"github.com/thanhftu/bookingservice/listener"
	"github.com/thanhftu/bookingservice/rest"
	"github.com/thanhftu/lib/configuration"
	"github.com/thanhftu/lib/msgqueue"
	msgqueue_amqp "github.com/thanhftu/lib/msgqueue/amqp"
	"github.com/thanhftu/lib/msgqueue/kafka"
	"github.com/thanhftu/lib/persistence/dblayer"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var eventListener msgqueue.EventListener
	var eventEmitter msgqueue.EventEmitter

	confPath := flag.String("conf", "./configuration/config.json", "flag to set the path to the configuration json file")
	flag.Parse()

	//extract configuration
	config, _ := configuration.ExtractConfiguration(*confPath)
	fmt.Println(config.MessageBrokerType)

	// we add this line to use kafka
	// fmt.Println(os.Getenv("KAFKA_BROKER_URLS"))
	// fmt.Println(config.MessageBrokerType)
	// fmt.Println(config.KafkaMessageBrokers)
	// config.MessageBrokerType = "kafka"

	switch config.MessageBrokerType {
	case "amqp":
		conn, err := amqp.Dial(config.AMQPMessageBroker)
		panicIfErr(err)

		eventListener, err = msgqueue_amqp.NewAMQPEventListener(conn, "events", "booking")
		panicIfErr(err)

		eventEmitter, err = msgqueue_amqp.NewAMQPEventEmitter(conn, "events")
		panicIfErr(err)
	case "kafka":
		conf := sarama.NewConfig()
		conf.Producer.Return.Successes = true
		conn, err := sarama.NewClient(config.KafkaMessageBrokers, conf)
		panicIfErr(err)

		eventListener, err = kafka.NewKafkaEventListener(conn, []int32{})
		panicIfErr(err)

		eventEmitter, err = kafka.NewKafkaEventEmitter(conn)
		panicIfErr(err)
	default:
		panic("Bad message broker type: " + config.MessageBrokerType)
	}

	dbhandler, _ := dblayer.NewPersistenceLayer(config.Databasetype, config.DBConnection)

	processor := listener.EventProcessor{eventListener, dbhandler}
	go processor.ProcessEvents()

	rest.ServeAPI(config.RestfulEndpoint, dbhandler, eventEmitter)
}
