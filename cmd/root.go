package cmd

import (
	generated "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/crypto-smoke/meshtastic-go/mqtt"
	"github.com/crypto-smoke/meshtastic-go/transport/serial"
	"google.golang.org/protobuf/proto"

	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type thing struct {
	*serial.Conn
	*mqtt.Client
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "meshtastic-client-proxy",
	Short: "A meshtastic mqtt client proxy via serial/USB",
	Long: `This will allow a USB or serial connected meshtastic node
to connect to MQTT via the host device's internet connection. 

NOTE: You must do the following in your device settings:
  1. Enable MQTT software module
  2. Enable MQTT client proxy
  3. Enable uplink and/or downlink on each channel you want to proxy

IMPORTANT: Currently this proxy ignores your device settings, so you
           must still specify the channels you want to proxy and the
           connection details for the MQTT broker.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		t := thing{}
		//log.SetLevel(log.DebugLevel)

		chans, _ := cmd.Flags().GetStringArray("channel")
		brokerURL, _ := cmd.Flags().GetString("broker-url")
		brokerUser, _ := cmd.Flags().GetString("user")
		brokerPass, _ := cmd.Flags().GetString("pass")
		rootTopic, _ := cmd.Flags().GetString("root")
		comPort, _ := cmd.Flags().GetString("port")

		client := mqtt.NewClient(brokerURL, brokerUser, brokerPass, rootTopic)
		err := t.ConnectMQTT(client, chans)
		if err != nil {
			log.Fatal("error connecting to mqtt broker", "err", err)
		}

		err = t.ConnectSerial(comPort, false)
		if err != nil {
			log.Fatal("error connecting to serial node", "err", err)
		}
		log.Info("serial connected")
		log.Info("started")
		select {}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.meshtastic-client-proxy.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("port", "p", "", "Serial port for node")

	rootCmd.Flags().String("broker-url", "tcp://mqtt.meshtastic.org:1883", "MQTT broker URL")
	rootCmd.Flags().String("user", "meshdev", "MQTT username")
	rootCmd.Flags().String("pass", "large4cats", "MQTT user password")
	rootCmd.Flags().StringP("root", "r", "msh", "MQTT root topic")

	rootCmd.Flags().StringArrayP("channel", "c", []string{"LongFast"}, "Channel(s) to proxy")

	cobra.MarkFlagRequired(rootCmd.Flags(), "port")
	cobra.MarkFlagRequired(rootCmd.Flags(), "channels")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".meshtastic-client-proxy" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".meshtastic-client-proxy")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func (t *thing) ConnectSerial(comport string, errorOnNoHandler bool) error {
	if comport == "" {
		potentialPorts := serial.GetPorts()
		if potentialPorts == nil {
			return errors.New("no usb serial devices detected")
		}
		if len(potentialPorts) > 1 {
			return errors.New("multiple ports detected")
		}

		comport = potentialPorts[0]
	}
	log.Info("Connecting to", "serial port", comport)
	sConn := serial.NewConn(comport, errorOnNoHandler)

	// TODO: I still dont like this method of registering handlers, but it's close and "good enough" for now
	// I would like to mirror how discordgo does it https://github.com/bwmarrin/Discordgo

	sConn.Handle(new(generated.FromRadio), func(msg proto.Message) {
		pkt := msg.(*generated.FromRadio)
		switch p := pkt.PayloadVariant.(type) {
		case *generated.FromRadio_MqttClientProxyMessage:
			// send to mqtt
			proxyMessage := p.MqttClientProxyMessage
			log.Info("mesh to mqtt", "topic", proxyMessage.Topic, "payload", hex.EncodeToString(proxyMessage.GetData()))
			mqttMessage := mqtt.Message{
				Topic:    proxyMessage.Topic,
				Payload:  proxyMessage.GetData(),
				Retained: proxyMessage.Retained,
			}
			err := t.Publish(&mqttMessage)
			if err != nil {
				log.Error("failed publishing message", "err", err)
				return
			}
			//chToMQTT <- &mqttMessage
			log.Info("message published to mqtt")
		}
	})
	err := sConn.Connect()
	if err != nil {
		return err
	}
	t.Conn = sConn
	return nil
}
func (t *thing) ConnectMQTT(client *mqtt.Client, channels []string) error {

	err := client.Connect()
	if err != nil {
		return err
	}
	for _, channel := range channels {
		client.Handle(channel, t.channelHandler(channel))
	}
	t.Client = client
	return nil
}

func (t *thing) channelHandler(channel string) mqtt.HandlerFunc {
	return func(m mqtt.Message) {
		var env generated.ServiceEnvelope
		err := proto.Unmarshal(m.Payload, &env)
		if err != nil {
			log.Error("failed unmarshalling to service envelope", "err", err, "payload", hex.EncodeToString(m.Payload))
			return
		}

		log.Info("got packet from mqtt", "topic", m.Topic, "channel", channel)

		toRadio := generated.ToRadio{
			PayloadVariant: &generated.ToRadio_MqttClientProxyMessage{
				MqttClientProxyMessage: &generated.MqttClientProxyMessage{
					Topic:          m.Topic,
					PayloadVariant: &generated.MqttClientProxyMessage_Data{Data: m.Payload},
					Retained:       m.Retained,
				},
			},
		}

		// send packet to radio over serial
		err = t.SendToRadio(&toRadio)
		if err != nil {
			log.Error("failed sending to radio", "err", err)
			return
		}
		log.Info("message sent to radio")
	}
}
