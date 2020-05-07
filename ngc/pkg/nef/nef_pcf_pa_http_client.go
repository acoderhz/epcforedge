/* SPDX-License-Identifier: Apache-2.0
* Copyright (c) 2020 Intel Corporation
 */

package ngcnef

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// PcfClient is an implementation of the Pcf Authorization
type PcfClient struct {
	Pcfcfg      *PcfPolicyAuthorizationConfig
	HTTPClient  *http.Client
	OAuth2Token string
	RootURI     string
	ResourceURI string
	UserAgent   string
}

const pdContentType string = "application/problem+json"

//HTTPclient creates a new HTTP Client
func genHTTPClient(cfg *Config) (*http.Client, error) {

	HTTPClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	if cfg.PcfPolicyAuthorizationConfig.Protocol == "https" {
		CACert, err1 := ioutil.ReadFile(cfg.PcfPolicyAuthorizationConfig.ClientCert)
		if err1 != nil {
			fmt.Printf("NEF Certification loading Error: %v", err1)
			return nil, err1

		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(CACert)
		var tlsConfig *tls.Config
		if !cfg.PcfPolicyAuthorizationConfig.VerifyCerts {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}

		} else {
			tlsConfig = &tls.Config{
				RootCAs: caCertPool,
			}

		}
		HTTPClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		if cfg.PcfPolicyAuthorizationConfig.ProtocolVer == "2.0" {

			HTTPClient.Transport = &http2.Transport{
				TLSClientConfig: tlsConfig,
			}
		}

	}
	return HTTPClient, nil
}

//NewPCFPolicyAuthHTTPClient creates a new PCF Client
func NewPCFPolicyAuthHTTPClient(cfg *Config) (*PcfClient, error) {

	c := &PcfClient{}
	var err error
	c.HTTPClient, err = genHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	c.Pcfcfg = cfg.PcfPolicyAuthorizationConfig
	base := cfg.PcfPolicyAuthorizationConfig.Protocol + "://" + cfg.PcfPolicyAuthorizationConfig.Hostname + ":"
	c.RootURI = base + cfg.PcfPolicyAuthorizationConfig.Port
	c.ResourceURI = cfg.PcfPolicyAuthorizationConfig.ResourceURI
	c.UserAgent = cfg.UserAgent
	log.Infoln("PCF Client created with the following configuration:")
	log.Infoln("Protocol: ", cfg.PcfPolicyAuthorizationConfig.Protocol)
	log.Infoln("Version: ", cfg.PcfPolicyAuthorizationConfig.ProtocolVer)
	log.Infoln("OAuth2Support: ", cfg.PcfPolicyAuthorizationConfig.OAuth2Support)
	log.Infoln("TLSVerify: ", cfg.PcfPolicyAuthorizationConfig.VerifyCerts)
	log.Infoln("Resource URI: ", c.RootURI+c.ResourceURI)
	return c, nil
}

//PolicyAuthorizationCreate is a actual implementation
// Successful response : 201 and body contains AppSessionContext
func (pcf *PcfClient) PolicyAuthorizationCreate(ctx context.Context,
	body AppSessionContext) (AppSessionID, PcfPolicyResponse, error) {

	log.Infof("PCFs PolicyAuthorizationCreate Entered")
	_ = ctx

	pcfPr := PcfPolicyResponse{}
	apiURL := pcf.RootURI + pcf.ResourceURI
	var appsesid string
	var req *http.Request
	var res *http.Response

	var appSessionContext AppSessionContext
	var problemDetails ProblemDetails
	appSessionID := AppSessionID("")
	var resbody []byte
	log.Infof("Triggering PCF Policy Authorization POST :" + apiURL)
	headerParams := make(map[string]string)
	headerParams["Content-Type"] = contentType
	headerParams["User-Agent"] = pcf.UserAgent
	postbody, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Failed marshaling POST body :%s", err)
		goto END

	}

	req, err = prepareRequest(ctx, apiURL, "POST", postbody,
		headerParams)
	if err != nil {
		goto END
	}

	res, err = pcf.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("Failed receiving POST response:%s", err)
		goto END
	}

	log.Infof("Body in the response =>")
	resbody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Failed reading POST response body:%s", err)
		goto END

	}
	log.Infof(string(resbody))

	defer closeRespBody(res)

	if res.StatusCode == 201 {
		appsesid = res.Header.Get("Location")
		log.Infof("appsessionid" + appsesid)
		appSessionID = AppSessionID(appsesid)
		appSessionContext = AppSessionContext{}
		err = json.Unmarshal(resbody, &appSessionContext)
		if err != nil {
			fmt.Printf("Failed unmarshaling POST response body:%s", err)
			goto END
		}
		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Asc = &appSessionContext
		pcfPr.Pd = nil

	} else {
		problemDetails = ProblemDetails{}
		respContentType := res.Header.Get("Content-type")
		if respContentType == pdContentType {
			e := json.Unmarshal(resbody, &problemDetails)
			if e != nil {
				fmt.Printf("Failed unmarshaling POST response body:%s", e)
				goto END
			}
		}
		log.Infof("PCFs PolicyAuthorizationCreate failed ")
		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Pd = &problemDetails
		if err == nil {
			err = errors.New("failed post")
		}

	}
END:
	return appSessionID, pcfPr, err

}

// PolicyAuthorizationUpdate is a actual implementation
// Successful response : 200 and body contains AppSessionContext
func (pcf *PcfClient) PolicyAuthorizationUpdate(ctx context.Context,
	body AppSessionContextUpdateData,
	appSessionID AppSessionID) (PcfPolicyResponse, error) {
	log.Infof("PCFs PolicyAuthorizationUpdate Entered for AppSessionID %s",
		string(appSessionID))
	_ = ctx

	pcfPr := PcfPolicyResponse{}
	sessid := string(appSessionID)
	apiURL := pcf.RootURI + pcf.ResourceURI + sessid
	var req *http.Request
	var res *http.Response
	var resbody []byte
	var appSessionContext AppSessionContext
	var problemDetails ProblemDetails

	fmt.Println(sessid)
	log.Infof("Triggering PCF Policy Authorization PATCH :" + apiURL)
	headerParams := make(map[string]string)
	headerParams["Content-Type"] = "application/merge-patch+json"
	headerParams["User-Agent"] = pcf.UserAgent
	patchbody, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Failed marshaling PATCH body:%s", err)
		goto END
	}
	req, err = prepareRequest(ctx, apiURL, "PATCH", patchbody,
		headerParams)
	if err != nil {
		goto END
	}

	res, err = pcf.HTTPClient.Do(req)

	if err != nil {
		fmt.Printf("Failed receiving PATCH response:%s", err)
		goto END
	}

	log.Infof("Body in the response =>")
	resbody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Failed reading PATCH response body:%s", err)
		goto END

	}
	log.Infof(string(resbody))

	appSessionID = AppSessionID(sessid)
	defer closeRespBody(res)

	if res.StatusCode == 200 {
		appSessionContext = AppSessionContext{}
		err = json.Unmarshal(resbody, &appSessionContext)
		if err != nil {
			fmt.Printf("Failed unmarshaling PATCH response body:%s", err)
			goto END
		}
		log.Infof("PCFs PolicyAuthorizationUpdate AppSessionID %s updated",
			string(appSessionID))

		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Asc = &appSessionContext

	} else {
		problemDetails = ProblemDetails{}
		respContentType := res.Header.Get("Content-type")
		if respContentType == pdContentType {
			err = json.Unmarshal(resbody, &problemDetails)
			if err != nil {
				fmt.Printf("Failed unmarshaling PATCH response body:%s", err)
				goto END
			}
		}
		log.Infof("PCFs PolicyAuthorizationUpdate AppSessionID %s not found",
			string(appSessionID))
		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Pd = &problemDetails
		if err == nil {
			err = errors.New("failed patch")
		}
	}

	log.Infof("PCFs PolicyAuthorizationUpdate Exited for AppSessionID %s",
		string(appSessionID))
END:
	return pcfPr, err
}

// PolicyAuthorizationDelete is a actual implementation
// Successful response : 204 and empty body
func (pcf *PcfClient) PolicyAuthorizationDelete(ctx context.Context,
	appSessionID AppSessionID) (PcfPolicyResponse, error) {

	log.Infof("PCFs PolicyAuthorizationDelete Entered for AppSessionID %s",
		string(appSessionID))
	_ = ctx

	pcfPr := PcfPolicyResponse{}
	sessid := string(appSessionID)
	var req *http.Request
	var res *http.Response
	var resbody []byte
	var err error
	apiURL := pcf.RootURI + pcf.ResourceURI + sessid + "/delete"

	log.Infof("Triggering PCF Policy Authorization Delete :" + apiURL)

	headerParams := make(map[string]string)
	headerParams["Content-Type"] = contentType
	headerParams["User-Agent"] = pcf.UserAgent

	req, err = prepareRequest(ctx, apiURL, "POST", nil,
		headerParams)
	if err != nil {
		goto END
	}

	res, err = pcf.HTTPClient.Do(req)

	if err != nil {
		fmt.Printf("Failed receiving DELETE response:%s", err)
		goto END
	}

	if res.StatusCode == 204 {
		log.Infof("PCFs PolicyAuthorizationDelete AppSessionID %s found",
			sessid)
		pcfPr.ResponseCode = uint16(res.StatusCode)

	} else if res.StatusCode == 200 {
		//var eventnoti EventsNotification
		log.Infof("Body in the response =>")
		resbody, err = ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("Failed reading DELETE response body:%s", err)
			goto END

		}
		defer closeRespBody(res)
		log.Infof(string(resbody))
		/* err = json.Unmarshal(body, &eventnoti)
		 if err != nil {
			 fmt.Printf("Failed go error :%s", err)
		 } */
		pcfPr.ResponseCode = uint16(res.StatusCode)

	} else {

		respContentType := res.Header.Get("Content-type")
		if respContentType == pdContentType {
			problemDetails := ProblemDetails{}
			err = json.Unmarshal(resbody, &problemDetails)
			if err != nil {
				fmt.Printf("Failed unmarshaling DELETE response body:%s", err)
				goto END
			}
			pcfPr.Pd = &problemDetails
		}
		log.Infof("PCFs PolicyAuthorizationDelete AppSessionID %s not found",
			sessid)
		if err == nil {
			err = errors.New("failed delete")
		}
		pcfPr.ResponseCode = uint16(res.StatusCode)
	}
	log.Infof("PCFs PolicyAuthorizationDelete Exited for AppSessionID %s",
		sessid)
END:
	return pcfPr, err
}

// PolicyAuthorizationGet is a actual implementation
// Successful response : 204 and empty body
func (pcf *PcfClient) PolicyAuthorizationGet(ctx context.Context,
	appSessionID AppSessionID) (PcfPolicyResponse, error) {
	log.Infof("PCFs PolicyAuthorizationGet Entered for AppSessionID %s",
		string(appSessionID))
	_ = ctx
	sessid := string(appSessionID)
	apiURL := pcf.RootURI + pcf.ResourceURI + sessid
	pcfPr := PcfPolicyResponse{}

	var res *http.Response
	var req *http.Request
	var appSessionContext AppSessionContext
	var problemDetails ProblemDetails
	var err error
	var resbody []byte
	log.Infof("Triggering PCF Policy Authorization GET : " + apiURL)
	headerParams := make(map[string]string)
	headerParams["Content-Type"] = contentType
	headerParams["User-Agent"] = pcf.UserAgent
	req, err = prepareRequest(ctx, apiURL, "GET", nil,
		headerParams)
	if err != nil {
		goto END
	}
	res, err = pcf.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("Failed creating GET response:%s", err)
		goto END

	}
	log.Infof("Body in the response =>")
	resbody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Failed reading GET response body:%s", err)
		goto END

	}
	log.Infof(string(resbody))

	defer closeRespBody(res)

	if res.StatusCode == 200 {
		appSessionContext = AppSessionContext{}
		err = json.Unmarshal(resbody, &appSessionContext)
		if err != nil {
			fmt.Printf("Failed unmarshaling GET response body:%s", err)
			goto END
		}
		log.Infof("PCFs PolicyAuthorizationGet AppSessionID %s found",
			string(appSessionID))

		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Asc = &appSessionContext

	} else {
		problemDetails = ProblemDetails{}
		respContentType := res.Header.Get("Content-type")
		if respContentType == pdContentType {
			err = json.Unmarshal(resbody, &problemDetails)
			if err != nil {
				fmt.Printf("Failed unmarshaling GET response body:%s", err)
				goto END
			}
		}
		log.Infof("PCFs PolicyAuthorizationGet AppSessionID %s not found",
			string(appSessionID))
		if err == nil {
			err = errors.New("failed get")
		}

		pcfPr.ResponseCode = uint16(res.StatusCode)
		pcfPr.Pd = &problemDetails
	}
END:
	return pcfPr, err
}
