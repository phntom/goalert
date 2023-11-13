package main

import "time"

const (
	//AlertsUrl     = "https://source-alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
	AlertsUrl     = "https://alerts.ynet.co.il/alertsRss/YnetPicodeHaorefAlertFiles.js?callback=jsonCallback"
	maxRetries    = 500                    // number of times to retry sending the message
	retryInterval = 200 * time.Millisecond // time between retries
	maxTimeout    = 90 * time.Second       // overall timeout for the entire process
	postTimeout   = 10 * time.Second
)
