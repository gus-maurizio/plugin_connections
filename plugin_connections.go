package main

import (
		"encoding/json"
		"fmt"
		"github.com/shirou/gopsutil/net"
		log "github.com/sirupsen/logrus"
		"github.com/prometheus/client_golang/prometheus"
		"github.com/prometheus/client_golang/prometheus/promhttp"
		"net/http"
    	"time"
)


var PluginConfig 	map[string]map[string]map[string]interface{}
var PluginData		map[string]interface{}


//	Define the metrics we wish to expose
var connMetrics = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "sreagent_conn_metrics",
		Help: "TCP connections metrics",
	}, []string{"status"} )



func PluginMeasure() ([]byte, []byte, float64) {

	var TCPStatuses = map[string]string{
	    "01": "ESTABLISHED",
	    "02": "SYN_SENT",
	    "03": "SYN_RECV",
	    "04": "FIN_WAIT1",
	    "05": "FIN_WAIT2",
	    "06": "TIME_WAIT",
	    "07": "CLOSE",
	    "08": "CLOSE_WAIT",
	    "09": "LAST_ACK",
	    "0A": "LISTEN",
	    "0B": "CLOSING",
	}
	connCounter := make(map[string]int,len(TCPStatuses))

	n, _			:= net.Connections("all")
	for _, status   := range TCPStatuses { connCounter[status] = 0 }
	for _, connStat := range n {
		if connStat.Status == "" { continue } 
		connCounter[connStat.Status] += 1 
	}

	PluginData["interfaces"], _	 = net.Interfaces()
	PluginData["conntotals"]     = connCounter
	PluginData["connections"]    = n

	// Update metrics related to the plugin
	for status, count   := range connCounter {
		if status == "" { continue }
		connMetrics.With(prometheus.Labels{"status": status}).Set(float64(count))
	}

	myMeasure, _ 	:= json.Marshal(PluginData)
	return myMeasure, []byte(""), float64(time.Now().UnixNano())/1e9
}


func InitPlugin(config string) () {
	if PluginData  		== nil {
		PluginData 		=  make(map[string]interface{},20)
	}
	if PluginConfig  	== nil {
		PluginConfig 	=  make(map[string]map[string]map[string]interface{},20)
	}


	// Register metrics with prometheus
	prometheus.MustRegister(connMetrics)

	err := json.Unmarshal([]byte(config), &PluginConfig)
	if err != nil {
		log.WithFields(log.Fields{"config": config}).Error("failed to unmarshal config")
	}

	log.WithFields(log.Fields{"pluginconfig": PluginConfig}).Info("InitPlugin")
}


func main() {
	config  := 	`{}`

	//--------------------------------------------------------------------------//
	// time to start a prometheus metrics server
	// and export any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(":8999", nil)
	}()
	//--------------------------------------------------------------------------//

	InitPlugin(config)
	log.WithFields(log.Fields{"PluginConfig": PluginConfig}).Info("InitPlugin")
	tickd := 2 * time.Second
	for i := 1; i <= 3; i++ {
		tick := time.Now().UnixNano()
		measure, measureraw, measuretimestamp := PluginMeasure()
		fmt.Printf("Iteration #%d tick %d \n", i, tick)
		log.WithFields(log.Fields{"timestamp": measuretimestamp, 
					  "measure": string(measure[:]),
					  "measureraw": string(measureraw[:]),
					  "PluginData": PluginData,
		}).Info("Tick")
		time.Sleep(tickd)
	}
}
