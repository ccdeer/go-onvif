package onvif

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/clbanning/mxj"
	uuid "github.com/satori/go.uuid"
)

var httpClient = &http.Client{Timeout: time.Second * 5}

// SOAP contains data for SOAP request
type SOAP struct {
	Body     string
	XMLNs    []string
	User     string
	Password string
	TokenAge time.Duration
	//authHeaders func(method string) []string
	AuthHeaders string
	URI         string
	Method      string
}

// SendRequest sends SOAP request to xAddr
func (soap *SOAP) SendRequest(xaddr string) (mxj.Map, error) {
	// Create SOAP request
	request := soap.createRequest()

	// Make sure URL valid and add authentication in xAddr
	urlXAddr, err := url.Parse(xaddr)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	buffer := bytes.NewBuffer([]byte(request))
	req, err := http.NewRequest("POST", urlXAddr.String(), buffer)
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("Charset", "utf-8")

	Debugf("[>>>%s]%s", xaddr, buffer.String())
	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Error(err)
		return nil, err
	}
	Debugf("[<<<%s]%s", xaddr, string(responseBody))

	if resp.StatusCode != 200 {

		if resp.StatusCode == 401 {
			fmt.Println("handle401")
			err = soap.handle401(resp)
			if err != nil {
				Error(err)
				return nil, err
			} else {

				buffer = bytes.NewBuffer([]byte(request))
				req, err = http.NewRequest("POST", urlXAddr.String(), buffer)
				req.Header.Set("Content-Type", "application/soap+xml")
				req.Header.Set("Charset", "utf-8")
				req.Header.Set("Authorization", string(soap.AuthHeaders))
				// Send request
				resp, err = httpClient.Do(req)
				if err != nil {
					Error(err)
					return nil, err
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					err = errors.New(resp.Status)
					Error(err)
					return nil, err
				}

			}

		} else {

			err = errors.New(resp.Status)
			Error(err)
			return nil, err
		}
	}

	// Parse XML to map
	mapXML, err := mxj.NewMapXml(responseBody)
	if err != nil {
		Error(err)
		return nil, err
	}
	//fmt.Println(mapXML)

	// Check if SOAP returns fault
	fault, _ := mapXML.ValueForPathString("Envelope.Body.Fault.Reason.Text.#text")
	if fault != "" {
		return nil, errors.New(fault)
	}

	return mapXML, nil
}

func (soap SOAP) createRequest() string {
	// Create request envelope
	request := `<?xml version="1.0" encoding="UTF-8"?>`
	request += `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">`

	// Set request header
	if soap.User != "" {
		request += "<s:Header>" + soap.createUserToken() + "</s:Header>"
	}

	// Set request body
	//Set XML namespace
	request += "<s:Body"
	for _, namespace := range soap.XMLNs {
		request += " " + namespace
	}
	request += ">"

	request += soap.Body + "</s:Body>"

	// Close request envelope
	request += "</s:Envelope>"

	// Clean request
	request = regexp.MustCompile(`\>\s+\<`).ReplaceAllString(request, "><")
	request = regexp.MustCompile(`\s+`).ReplaceAllString(request, " ")

	return request
}

func (soap SOAP) createUserToken() string {
	//nonce := uuid.NewV4().Bytes()

	id := uuid.NewV4()

	nonce := id.Bytes()

	nonce64 := base64.StdEncoding.EncodeToString(nonce)
	timestamp := time.Now().Add(soap.TokenAge).UTC().Format(time.RFC3339)
	token := string(nonce) + timestamp + soap.Password

	sha := sha1.New()
	sha.Write([]byte(token))
	shaToken := sha.Sum(nil)
	shaDigest64 := base64.StdEncoding.EncodeToString(shaToken)

	return `<Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
  		<UsernameToken>
    		<Username>` + soap.User + `</Username>
    		<Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">` + shaDigest64 + `</Password>
    		<Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">` + nonce64 + `</Nonce>
    		<Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">` + timestamp + `</Created>
		</UsernameToken>
	</Security>`
}

func md5hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
func (soap *SOAP) handle401(res *http.Response) (err error) {
	authval := res.Header.Get("WWW-Authenticate")
	hdrval := strings.SplitN(authval, " ", 2)
	fmt.Println("hdrval:", hdrval)
	var realm, qop, nonce string

	if len(hdrval) == 2 {
		for _, field := range strings.Split(hdrval[1], ",") {
			field = strings.Trim(field, ", ")
			if keyval := strings.Split(field, "="); len(keyval) == 2 {
				key := keyval[0]
				val := strings.Trim(keyval[1], `"`)
				switch key {
				case "realm":
					realm = val
				case "nonce":
					nonce = val
				case "qop":
					qop = val
				}
			}
		}

		fmt.Println("realm,nonce,qop", realm, nonce, qop)

		if realm != "" {
			var username string
			var password string

			if soap.User == "" {
				err = fmt.Errorf("no username")
				return
			}
			username = soap.User
			password = soap.Password

			soap.AuthHeaders = fmt.Sprintf(`Basic %s`, base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

			if nonce != "" {
				var response string

				id := uuid.NewV4()
				cnonce := md5hash(string(id.Bytes()))
				fmt.Println("cnonce:", cnonce)

				nc := "00000001"
				a1 := username + ":" + realm + ":" + password

				a2 := soap.Method + ":" + soap.URI
				hs1 := md5hash(a1)

				switch qop {
				case "auth-int":
					hs2 := md5hash(a2 + ":" + md5hash("entityBody"))
					response = md5hash(hs1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + hs2)
				case "auth":
					hs2 := md5hash(a2)
					response = md5hash(hs1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + hs2)
				default:
					hs2 := md5hash(a2)
					response = md5hash(hs1 + ":" + nonce + ":" + hs2)
				}

				soap.AuthHeaders = fmt.Sprintf(
					`Digest username="%s", realm="%s", qop="%s", algorithm="MD5", uri="%s", nonce="%s", nc=%s, cnonce="%s", opaque="", response="%s"`,
					username, realm, qop, soap.URI, nonce, nc, cnonce, response)

			}

		}
		//fmt.Println("soap.AuthHeaders:", soap.AuthHeaders)

	} else {
		Error("no username")
	}
	return
}
