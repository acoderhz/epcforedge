// Copyright 2019 Intel Corporation and Smart-Edge.com, Inc. All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/otcshare/edgenode/pkg/config"
)

//HTTPConfig : Contains the configuration for the HTTP 1.1
type HttpConfig struct {
	Endpoint                 string `json:"endpoint"`,	
}

//HTTP2Config : Contains the configuration for the HTTP2 
type Http2Config struct {
	Endpoint                 string `json:"endpoint"`,
	NefServerCert			 string `json:"NefServerCert"`,
	NefServerKey			 string `json:"NefServerKey"`,
	AfServerCert			 string `json:"AfServerCert"`
}

// Config: NEF Module Configuration Data Structure
type Config struct {
	LocationPrefix            string `json:"locationPrefix"`
	MaxSubSupport             int    `json:"maxSubSupport"`
	MaxAFSupport              int    `json:"maxAFSupport"`
	SubStartID                int    `json:"subStartID"`
	UpfNotificationResUriPath string `json:"UpfNotificationResUriPath"`
	UserAgent                 string `json:"UserAgent"`
	HttpConfig				  HttpConfig
	Http2Config				  Http2Config

}

// NEF Module Context Data Structure
type nefContext struct {
	cfg Config
	nef nefData
}

// runServer : This function cretaes a Router object and starts a HTTP Server
//             in a separate go routine. Also it listens for NEF module
//             running context cancellation event in another go routine. If
//             cancellation event occurs, it shutdowns the HTTP Server.
// Input Args:
//   - ctx:    NEF Module Running context
//   - nefCtx: This is NEF Module Context. This contains the NEF Module Data.
// Output Args:
//    - error: retruns no error. It only logs the error if any happened while
//             starting the HTTP Server
func runServer(ctx context.Context, nefCtx *nefContext) error {

	var err error

	/* NEFRouter obeject is created. After creation this object contains all
	 * the HTTP Service Handlers. These hanlders will be called when HTTP
	 * server receives any HTTP Request */
	nefRouter := NewNEFRouter(nefCtx)

	// 1 for http2, 1 for http and 1 for the os signal
	numchannels := 3

	// Check if http and http 2 are both configured to determine number 
	// of channels

	if nefCtx.cfg.HttpConfig.Endpoint == nil {
		log.Info("HTTP Server not configured")
		numchannels--		
	} else {
		// HTTP Server object is created
		server := &http.Server {
			Addr:           nefCtx.cfg.Endpoint,
			Handler:        nefRouter,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
	}	

	if nefCtx.cfg.Http2Config.Endpoint == nil {
		log.Info("HTTP 2 Server not configured")
		numchannels--		
	} else {
		serverHttp2 := &http.Server{
			Addr:           nefCtx.cfg.Endpoint,
			Handler:        nefRouter,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	
		if err = http2.ConfigureServer(serverHttp2, &http2.Server{}); err != nil {
			log.Errf("NEF failed at configuring HTTP2 server")
			return err
		}	
	}	

	stopServerCh := make(chan bool, numchannels)

	/* Go Routine is spawned here for listening for cancellation event on
	 * context */
	go func(stopServerCh chan bool) {
		<-ctx.Done()
		log.Info("Executing graceful stop for NEF HTTP Server")
		if err = server.Close(); err != nil {
			log.Errf("Could not close NEF HTTP server: %#v", err)
		}
		log.Info("NEF HTTP server stopped")

		log.Info("Executing graceful stop for NEF HTTP2 Server")
		if err = serverHttp2.Close(); err != nil {
			log.Errf("Could not close NEF HTTP2 server: %#v", err)
		}
		log.Info("NEF HTTP2 server stopped")

		/* De-initializes NEF Data */
		nefCtx.nef.nefDestroy()

		stopServerCh <- true
	}(stopServerCh)

	/* Go Routine is spawned here for starting HTTP Server */
	go func(stopServerCh chan bool) {
		if nefCtx.cfg.HttpConfig.Endpoint != nil {
			log.Infof("NEF HTTP 1.1 listening on %s", server.Addr)
			if err = server.ListenAndServe(); err != nil {
				log.Errf("NEF server error: " + err.Error())
			}
		stopServerCh <- true
	}(stopServerCh)

	/* Go Routine is spawned here for starting HTTP-2 Server */
	go func(stopServerCh chan bool) {
		if nefCtx.cfg.HttpConfig2.Endpoint != nil {
			log.Infof("NEF HTTP 2.0 listening on %s", server.Addr)
			if err = serverHttp2.ListenAndServeTLS(
				nefCtx.cfg.SrvCfg.Http2Config.NefServerCert,
				nefCtx.cfg.SrvCfg.Http2Config.NefServerKey); err != nil {
				log.Errf("NEF server error: " + err.Error())
			}
			log.Info("Exiting")
		}
		stopServerCh <- true
	}(stopServerCh)

	/* This self go routine is waiting for the receive events from the spawned
	 * go routines */
	<-stopServerCh
	<-stopServerCh
	if numchannels == 3 
		<-stopServerCh
	return nil
}

// Run : This function reads the NEF Module configuration file and stores in
//       NEF Module Context. This also calls the Initialization/Creation of
//       NEF Data. Also it  calls runServer function for starting HTTP Server.
// Input Args:
//    - ctx:     NEF Module Running context
//    - cfgPath: This is NEF Module Configuration file path
// Output Args:
//     - error: returns error in case any error occurred in reading NEF
//              configuration file, NEF create error or any error occurred in
//              starting server
func Run(ctx context.Context, cfgPath string) error {

	var nefCtx nefContext

	/* Reads NEF Configuration file which is json format. Also it converts
	 * configuration data from json format to structure data */
	err := config.LoadJSONConfig(cfgPath, &nefCtx.cfg)
	if err != nil {
		log.Errf("Failed to load NEF configuration: %v", err)
		return err
	}

	/* Creates/Initializes NEF Data */
	err = nefCtx.nef.nefCreate()
	if err != nil {
		log.Errf("NEF Create Failed: %v", err)
		return err
	}

	return runServer(ctx, &nefCtx)
}
