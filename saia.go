/*Package saia provides tooling to connect to the SAIA Trucking API.  This is for truck shipments,
not small parcels.  Think LTL (less than truckload) shipments.  This code was created off the SAIA API
documentation.  This uses and XML SOAP API.

You will need to have a SAIA Secure account and register for access to use this.

Currently this package can perform:
- pickup requests

To create a pickup request:
- Set test or production mode (SetProductionMode()).
- Set shipper information.
- Set shipment data.
- Request the pickup (RequestPickup()).
- Check for any errors.
*/
package saia

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//api url
const saiaURL = "http://www.saiasecure.com/webservice/pickup/xml.aspx"

//test mode
//this is set as an attribute in the xml request
//values are either "Y" or "N"
//this can be updated using the SetProductionMode() func.
var testMode = "Y"

//timeout is the default time we should wait for a reply from Ward
//You may need to adjust this based on how slow connecting to Ward is for you.
//10 seconds is overly long, but sometimes Ward is very slow.
var timeout = time.Duration(10 * time.Second)

//base XML data
//Don't need below stuff....SAIA docs are very confusing.  Example code shows everything
//is based of of the "Create" xml node.
/*
const (
	xsiAttr  = "http://www.w3.org/2001/XMLSchema-instance"
	xsdAttr  = "http://www.w3.org/2001/XMLSchema"
	soapAttr = "http://schemas.xmlsoap.org/soap/envelope/"

	createXmlnsAttr = "http://www.SaiaSecure.com/WebService/Pickup"
)

//PickupRequest is the main body of the xml request
type PickupRequest struct {
	XMLName xml.Name `xml:"soap:Envelope"`

	XsiAttr  string `xml:"xmlns:xsi,attr"`  //"http://www.w3.org/2001/XMLSchema-instance"
	XsdAttr  string `xml:"xmlns:xsd,attr"`  //"http://www.w3.org/2001/XMLSchema"
	SoapAttr string `xml:"xmlns:soap,attr"` //"http://schemas.xmlsoap.org/soap/envelope/"

	Create Create `xml:"soap:Body>Create"`
}

//Create is a container for a request
type Create struct {
	XmlnsAttr string `xml:"xmlns,attr"` //http://www.SaiaSecure.com/WebService/Pickup

	Request Request `xml:"soap:Body>Create>request"`
}
*/

//Request is the pickup request data
type Request struct {
	XMLName xml.Name `xml:"Create"`

	//required
	Item          Item   `xml:"Details>DetailItem"` //shipment details
	UserID        string //saia secure
	Password      string //saia secure
	TestMode      string //Y or N
	AccountNumber string //shipper's saia account number

	//optional
	CompanyName         string //pickup location company name, can be left empty is AccountNumber is for the pickup location
	Street              string
	City                string
	State               string //two character code
	Zipcode             string
	ContactName         string //shipping department or person's name
	ContactPhone        string
	PickupDate          string //yyyy-mm-dd
	ReadyTime           string //hh:mm:ss, 24 hour time
	CloseTime           string //hh:mm:ss, 24 hour time
	SpecialInstructions string
}

//Item is details on the shipment
type Item struct {
	//required
	DestinationZipcode string
	Pieces             uint
	Package            string  //two character code, SK = skids
	Weight             float64 //lbs

	//optional
	DestinationCountry string //US, CN, MX
	Freezable          string //Y or N
}

//ResponseData is data returned from a pickup request
//handles successful and errors
type ResponseData struct {
	Code            string
	Element         string
	Fault           string //S = server, C = client
	Message         string
	TestMode        string //Y or N
	PickupNumber    string //pickup confirmation number
	TotalPieces     uint
	TotalWeight     float64
	NextBusinessDay string //only set when code=S04, pickup cannot be made today and should be rescheduled for tomorrow
	PickupTerminal  PickupTerminal
}

//PickupTerminal is data on the terminal making a pickup
type PickupTerminal struct {
	ID                   string
	Name                 string
	Manager              string
	Address1             string
	Address2             string
	City                 string
	State                string
	Zipcode              string
	CityDispatchPhone    string
	CustomerServicePhone string
	TollFreePhone        string
	Fax                  string
}

//SetProductionMode chooses the production url for use
func SetProductionMode(yes bool) {
	if yes {
		testMode = "N"
	}
	return
}

//SetTimeout updates the timeout value to something the user sets
//use this to increase the timeout if connecting to saia is really slow
func SetTimeout(seconds time.Duration) {
	timeout = time.Duration(seconds * time.Second)
	return
}

//RequestPickup performs the call to the saia API to schedule a pickup
func (p *Request) RequestPickup() (responseData ResponseData, err error) {
	//set test mode flag as needed
	p.TestMode = testMode

	//convert to xml
	xmlBytes, err := xml.Marshal(p)
	if err != nil {
		err = errors.Wrap(err, "saia.RequestPickup - could not marshal xml")
		return
	}

	//add the xml header
	xmlString := xml.Header + string(xmlBytes)
	log.Print(xmlString)

	//make the call to the saia API
	//set a timeout since golang doesn't set one by default and we don't want this to hang forever
	httpClient := http.Client{
		Timeout: timeout,
	}
	res, err := httpClient.Post(saiaURL, "text/xml", strings.NewReader(xmlString))
	if err != nil {
		err = errors.Wrap(err, "saia.RequestPickup - could not make post request")
		return
	}

	//read the response
	//response should hold success or error data
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		err = errors.Wrap(err, "saia.RequestPickup - could not read response 1")
		return
	}

	err = xml.Unmarshal(body, &responseData)
	if err != nil {
		err = errors.Wrap(err, "saia.RequestPickup - could not read response 2")
		return
	}

	if responseData.Code != "" || responseData.PickupNumber == "" {
		log.Println("saia.RequestPickup - pickup request failed")
		log.Printf("%+v", responseData)

		err = errors.New("saia.RequestPickup - pickup request failed")
		err = errors.Wrap(err, responseData.Message)
		return
	}

	log.Printf("%+v", responseData)

	//pickup request successful
	//response data will have confirmation info
	return
}
