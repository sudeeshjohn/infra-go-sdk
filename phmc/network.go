package hmc

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// GetVirtualSwitchQuickAll retrieves a JSON array of all Virtual Switches on a Managed System.
func (c *RestClient) GetVirtualSwitchQuickAll(ctx context.Context, sysUUID string) ([]VirtualSwitchQuick, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/quick/All", c.hmcIP, sysUUID)


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/json")
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Switches Quick/All: %s", resp.Status)
	}

	var switches []VirtualSwitchQuick
	if err := json.Unmarshal(body, &switches); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}


	return switches, nil
}

// GetVirtualSwitchQuick retrieves JSON details for a specific Virtual Switch.
func (c *RestClient) GetVirtualSwitchQuick(ctx context.Context, sysUUID, switchUUID string) (*VirtualSwitchQuick, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s/quick", c.hmcIP, sysUUID, switchUUID)


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/json")

	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Switch Quick: %s", resp.Status)
	}

	var vSwitch VirtualSwitchQuick
	if err := json.Unmarshal(body, &vSwitch); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	// The singular /quick endpoint omits the UUID, so we inject it manually
	vSwitch.UUID = switchUUID


	return &vSwitch, nil
}

// GetVirtualSwitches retrieves the comprehensive XML feed of Virtual Switches on a Managed System.
func (c *RestClient) GetVirtualSwitches(ctx context.Context, sysUUID string) ([]VirtualSwitch, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch", c.hmcIP, sysUUID)


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")

	timeoutCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Switches XML: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, err
	}

	var switches []VirtualSwitch

	// Loop through every <entry> in the Atom feed
	for _, entry := range doc.FindElements("//entry") {
		vSwitch := VirtualSwitch{}

		// Extract Core properties
		if atomID := entry.FindElement(".//AtomID"); atomID != nil {
			vSwitch.UUID = atomID.Text()
		}
		if switchID := entry.FindElement(".//SwitchID"); switchID != nil {
			vSwitch.SwitchID = switchID.Text()
		}
		if switchName := entry.FindElement(".//SwitchName"); switchName != nil {
			vSwitch.SwitchName = switchName.Text()
		}
		if switchMode := entry.FindElement(".//SwitchMode"); switchMode != nil {
			vSwitch.SwitchMode = switchMode.Text()
		}

		// Extract VirtualNetwork Links
		for _, link := range entry.FindElements(".//VirtualNetworks/link") {
			if href := link.SelectAttrValue("href", ""); href != "" {
				vSwitch.VirtualNetworks = append(vSwitch.VirtualNetworks, href)
			}
		}

		switches = append(switches, vSwitch)
	}


	return switches, nil
}

// GetClientNetworkAdapters retrieves all ClientNetworkAdapter details for a partition as parsed Go structs.
func (c *RestClient) GetClientNetworkAdapters(ctx context.Context, systemUUID, lparUUID string) ([]ClientNetworkAdapter, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/LogicalPartition/%s/ClientNetworkAdapter", c.hmcIP, systemUUID, lparUUID)


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")

	timeoutCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}


	// ✨ THE FIX: Handle the 204 No Content status cleanly
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNoContent {
			return []ClientNetworkAdapter{}, nil
		}
		return nil, fmt.Errorf("request failed with status %s", resp.Status)
	}

	// Parse XML response and strip namespaces
	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces from XML: %v", err)
	}

	var adapters []ClientNetworkAdapter

	// Extract the core elements and natively unmarshal them into structs
	adapterElements := doc.FindElements("//ClientNetworkAdapter")

	for _, elem := range adapterElements {
		adapterDoc := etree.NewDocument()
		adapterDoc.SetRoot(elem.Copy())
		adapterBytes, err := adapterDoc.WriteToBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize adapter element: %v", err)
		}

		var adapter ClientNetworkAdapter
		if err := xml.Unmarshal(adapterBytes, &adapter); err != nil {
			return nil, fmt.Errorf("failed to unmarshal adapter XML: %v", err)
		}

		adapters = append(adapters, adapter)
	}


	return adapters, nil
}

// GetNetworkBootDevicesForLpar retrieves network boot devices from an LPAR's profile using the HMC REST API job.
//
// WARNING: This operation will power off the LPAR if it is currently running.
// The HMC GetNetworkBootDevices job requires the LPAR to be in a powered-off state
// to retrieve accurate network boot device information from the profile.
//
// Parameters:
//   - lparUUID: UUID of the logical partition
//   - profileUUID: UUID of the partition profile to query
//   - debug: Enable detailed logging
//
// Returns:
//   - []NetworkBootDevice: List of network boot devices configured in the profile
//   - error: Error if the operation fails
//
// Reference: HMC REST API - GetNetworkBootDevices job operation
func (c *RestClient) GetNetworkBootDevicesForLpar(ctx context.Context, lparUUID, profileUUID string) ([]NetworkBootDevice, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/LogicalPartition/%s/do/GetNetworkBootDevices", c.hmcIP, lparUUID)


	// Define operation details for the JobRequest
	reqdOperation := map[string]string{
		"OperationName": "GetNetworkBootDevices",
		"GroupName":     "LogicalPartition",
		"ProgressType":  "DISCRETE",
	}

	// Build job parameters
	jobParams := map[string]string{
		"LogicalPartitionProfileUUID": profileUUID,
	}

	// Create XML payload using the existing job request helper
	payload, err := createJobRequestPayload(reqdOperation, jobParams, "V1_1_0", true)
	if err != nil {
		return nil, fmt.Errorf("failed to create job request payload: %v", err)
	}

	// Create and execute HTTP request
	req, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Content-Type", "application/vnd.ibm.powervm.web+xml;type=JobRequest")
	req.Header.Set("Accept", "*/*")

	timeoutCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}


	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(respBody))
	}

	// Parse response to extract JobID
	doc, err := xmlStripNamespace(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces from XML: %v", err)
	}

	jobIDElem := doc.FindElement("//JobID")
	if jobIDElem == nil {
		return nil, fmt.Errorf("JobID not found in response")
	}
	jobID := jobIDElem.Text()


	// Wait for job completion
	jobResp, err := c.FetchJobStatus(ctx, jobID, false, 5)
	if err != nil {
		return nil, fmt.Errorf("GetNetworkBootDevices job failed: %v", err)
	}

	// Parse job results to extract network boot devices
	var bootDevices []NetworkBootDevice

	for _, param := range jobResp.Results.Parameters {
		if param.ParameterName == "result" {
			xmlContent := param.ParameterValue

			// 1. Clean the CDATA wrapper and unescape any HTML entities
			xmlContent = strings.TrimSpace(xmlContent)
			xmlContent = strings.TrimPrefix(xmlContent, "<![CDATA[")
			xmlContent = strings.TrimSuffix(xmlContent, "]]>")
			xmlContent = html.UnescapeString(strings.TrimSpace(xmlContent))


			// 2. Strip Namespaces to make Unmarshaling easy
			cleanDoc, err := xmlStripNamespace([]byte(xmlContent))
			if err != nil {
				continue
			}
			strippedXML, _ := cleanDoc.WriteToBytes()

			// 3. Use an inline struct to map IBM's XML directly using standard encoding/xml
			var collection struct {
				Devices []struct {
					BootDevice       string `xml:"BootDevice"`
					IsPhysicalDevice bool   `xml:"IsPhysicalDevice"`
					LocationCode     string `xml:"LocationCode"`
					MACAddressValue  string `xml:"MACAddressValue"`
				} `xml:"NetworkBootDevice"`
			}

			if err := xml.Unmarshal(strippedXML, &collection); err != nil {
				continue
			}

			// 4. Map the unmarshaled data to your SDK's return struct
			for _, d := range collection.Devices {
				deviceType := "virtual"
				if d.IsPhysicalDevice {
					deviceType = "physical"
				}

				bootDevices = append(bootDevices, NetworkBootDevice{
					DeviceName:   d.BootDevice,
					DeviceType:   deviceType,
					LocationCode: d.LocationCode,
					MACAddress:   d.MACAddressValue,
				})
			}
		}
	}


	return bootDevices, nil
}

// GetNetworkBootDevicesForVios retrieves the network boot devices of a VIOS
// partition using the HMC "GetNetworkBootDevices" VirtualIOServer job.
//
// The HMC runs lpar_netboot behind the scenes: it powers the partition on to
// Open Firmware, collects the adapter location codes / MAC addresses and
// closes the console again. The partition must therefore NOT be running.
// Use GetNetworkBootDevicesForViosImmediate to also force a running VIOS down.
//
// Note: the job does not accept LogicalPartitionProfileUUID for VIOS targets;
// it is parameterized by names (see the sibling Immediate variant).
//
// Parameters:
//   - viosUUID: UUID of the Virtual I/O Server (used in the endpoint URL)
//   - sysName: managed system name (e.g. "LTC23U01-JEMT-P11")
//   - viosName: VIOS partition name (e.g. "ltc23u01-vios1")
//   - profileName: partition profile name (e.g. "default_profile")
//   - loggedInUser: HMC user submitting the job (e.g. "hscroot")
func (c *RestClient) GetNetworkBootDevicesForVios(ctx context.Context, viosUUID, sysName, viosName, profileName, loggedInUser string) ([]NetworkBootDevice, error) {
	return c.getViosNetworkBootDevices(ctx, viosUUID, sysName, viosName, profileName, loggedInUser, false)
}

// GetNetworkBootDevicesForViosImmediate is like GetNetworkBootDevicesForVios
// but sets the "withImmediate" job parameter to true, which makes the HMC shut
// the VIOS down immediately if it is running.
//
// WARNING: this is disruptive. A running VIOS is powered off immediately,
// without a graceful OS shutdown.
func (c *RestClient) GetNetworkBootDevicesForViosImmediate(ctx context.Context, viosUUID, sysName, viosName, profileName, loggedInUser string) ([]NetworkBootDevice, error) {
	return c.getViosNetworkBootDevices(ctx, viosUUID, sysName, viosName, profileName, loggedInUser, true)
}

// buildViosNetworkBootDevicesPayload builds the JobRequest body for the VIOS
// GetNetworkBootDevices job. All values are XML-escaped.
func buildViosNetworkBootDevicesPayload(sysName, viosName, profileName, loggedInUser string, immediate bool) string {
	params := []struct{ name, value string }{
		{"managedSystemName", sysName},
		{"laparName", viosName}, // sic: the HMC parameter is spelled "laparName"
		{"profileName", profileName},
		{"loggedinuser", loggedInUser},
	}
	if immediate {
		params = append(params, struct{ name, value string }{"withImmediate", "true"})
	}

	var sb strings.Builder
	sb.WriteString(`<JobRequest
  xmlns="http://www.ibm.com/xmlns/systems/power/firmware/web/mc/2012_10/"
  xmlns:ns2="http://www.w3.org/XML/1998/namespace/k2" schemaVersion="V1_0">
  <Metadata>
    <Atom/>
  </Metadata>
  <RequestedOperation kxe="false" kb="CUR" schemaVersion="V1_0">
    <Metadata>
      <Atom/>
    </Metadata>
    <OperationName kxe="false" kb="ROR">GetNetworkBootDevices</OperationName>
    <GroupName kxe="false" kb="ROR">VirtualIOServer</GroupName>
  </RequestedOperation>
  <JobParameters kxe="false" kb="CUR" schemaVersion="V1_0">
    <Metadata>
      <Atom/>
    </Metadata>
`)
	for _, p := range params {
		fmt.Fprintf(&sb, `    <JobParameter schemaVersion="V1_0">
      <Metadata>
        <Atom/>
      </Metadata>
      <ParameterName kb="ROR" kxe="false">%s</ParameterName>
      <ParameterValue kxe="false" kb="CUR">%s</ParameterValue>
    </JobParameter>
`, p.name, html.EscapeString(p.value))
	}
	sb.WriteString("  </JobParameters>\n</JobRequest>")
	return sb.String()
}

// getViosNetworkBootDevices submits the VIOS GetNetworkBootDevices job, waits
// for completion and parses the resulting NetworkBootDevice_Collection.
func (c *RestClient) getViosNetworkBootDevices(ctx context.Context, viosUUID, sysName, viosName, profileName, loggedInUser string, immediate bool) ([]NetworkBootDevice, error) {
	if viosUUID == "" || sysName == "" || viosName == "" || profileName == "" || loggedInUser == "" {
		return nil, fmt.Errorf("viosUUID, sysName, viosName, profileName and loggedInUser are all required")
	}

	url := fmt.Sprintf("https://%s/rest/api/uom/VirtualIOServer/%s/do/GetNetworkBootDevices", c.hmcIP, viosUUID)
	payload := buildViosNetworkBootDevicesPayload(sysName, viosName, profileName, loggedInUser, immediate)

	timeoutCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Content-Type", "application/vnd.ibm.powervm.web+xml; type=JobRequest")
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(respBody))
	}

	doc, err := xmlStripNamespace(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces from XML: %v", err)
	}
	jobIDElem := doc.FindElement("//JobID")
	if jobIDElem == nil || strings.TrimSpace(jobIDElem.Text()) == "" {
		return nil, fmt.Errorf("JobID not found in response")
	}
	jobID := strings.TrimSpace(jobIDElem.Text())

	// The job power-cycles the partition to Open Firmware; it typically takes ~1 minute.
	jobResp, err := c.FetchJobStatus(ctx, jobID, false, 5)
	if err != nil {
		return nil, fmt.Errorf("GetNetworkBootDevices job failed: %v", err)
	}

	return parseNetworkBootDevicesResult(jobResp)
}

// parseNetworkBootDevicesResult extracts the NetworkBootDevice_Collection that
// the HMC returns, XML-escaped, in the job's "result" parameter.
func parseNetworkBootDevicesResult(jobResp *JobResponse) ([]NetworkBootDevice, error) {
	for _, param := range jobResp.Results.Parameters {
		if param.ParameterName != "result" {
			continue
		}

		xmlContent := strings.TrimSpace(param.ParameterValue)
		xmlContent = strings.TrimPrefix(xmlContent, "<![CDATA[")
		xmlContent = strings.TrimSuffix(xmlContent, "]]>")
		xmlContent = html.UnescapeString(strings.TrimSpace(xmlContent))
		if xmlContent == "" {
			return nil, nil
		}

		cleanDoc, err := xmlStripNamespace([]byte(xmlContent))
		if err != nil {
			return nil, fmt.Errorf("failed to strip namespaces from result XML: %v", err)
		}
		strippedXML, err := cleanDoc.WriteToBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize result XML: %v", err)
		}

		var collection NetworkBootDeviceCollection
		if err := xml.Unmarshal(strippedXML, &collection); err != nil {
			return nil, fmt.Errorf("failed to unmarshal NetworkBootDevice_Collection: %v", err)
		}

		devices := make([]NetworkBootDevice, 0, len(collection.Devices))
		for _, d := range collection.Devices {
			deviceType := "virtual"
			if strings.EqualFold(strings.TrimSpace(d.IsPhysicalDevice), "true") {
				deviceType = "physical"
			}
			devices = append(devices, NetworkBootDevice{
				DeviceName:   strings.TrimSpace(d.BootDevice),
				DeviceType:   deviceType,
				LocationCode: strings.TrimSpace(d.LocationCode),
				MACAddress:   strings.TrimSpace(d.MACAddressValue),
			})
		}
		return devices, nil
	}
	return nil, nil
}




// CreateClientNetworkAdapter adds a new Virtual Ethernet Adapter to an LPAR and connects it to a Virtual Switch.
// Returns the complete ClientNetworkAdapter structure with all details.
func (c *RestClient) CreateClientNetworkAdapter(ctx context.Context, sysUUID, lparUUID, vswitchUUID string, vlanID int) (*ClientNetworkAdapter, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/LogicalPartition/%s/ClientNetworkAdapter", c.hmcIP, lparUUID)


	// Fix: Changed <VirtualSwitch> to <AssociatedVirtualSwitch> with a <link> child.
	payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ClientNetworkAdapter:ClientNetworkAdapter xmlns:ClientNetworkAdapter="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                                           xmlns="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                                           schemaVersion="V1_0">
    <Metadata><Atom/></Metadata>
    <PortVLANID>%d</PortVLANID>
    <AssociatedVirtualSwitch>
        <link href="https://%s:443/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s" rel="related"/>
    </AssociatedVirtualSwitch>
</ClientNetworkAdapter:ClientNetworkAdapter>`,
		vlanID, c.hmcIP, sysUUID, vswitchUUID)

	httpReq, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Session", c.session)
	httpReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=ClientNetworkAdapter")
	httpReq.Header.Set("Accept", "application/atom+xml")

	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	httpReq = httpReq.WithContext(timeoutCtx)


	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Adapter creation failed (%s)", resp.Status)
	}

	// Parse the XML response and strip namespaces
	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces from XML: %v", err)
	}

	// Extract the ClientNetworkAdapter element
	adapterElement := doc.FindElement("//ClientNetworkAdapter")
	if adapterElement == nil {
		return nil, fmt.Errorf("ClientNetworkAdapter element not found in response")
	}

	// Convert the element to a standalone document for unmarshaling
	adapterDoc := etree.NewDocument()
	adapterDoc.SetRoot(adapterElement.Copy())
	adapterBytes, err := adapterDoc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize adapter element: %v", err)
	}

	// Unmarshal into the ClientNetworkAdapter struct
	var adapter ClientNetworkAdapter
	if err := xml.Unmarshal(adapterBytes, &adapter); err != nil {
		return nil, fmt.Errorf("failed to unmarshal adapter XML: %v", err)
	}


	return &adapter, nil
}

// DeleteClientNetworkAdapter removes a specific Virtual Ethernet Adapter from an LPAR.
func (c *RestClient) DeleteClientNetworkAdapter(ctx context.Context, lparUUID, adapterUUID string) error {
	// The specific endpoint for the adapter we want to delete
	url := fmt.Sprintf("https://%s/rest/api/uom/LogicalPartition/%s/ClientNetworkAdapter/%s", c.hmcIP, lparUUID, adapterUUID)


	// Create the DELETE request (No XML body required!)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")

	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req = req.WithContext(timeoutCtx)


	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// A successful DELETE operation typically returns 200 OK or 204 No Content
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete adapter (%s): %s", resp.Status, string(body))
	}


	return nil
}
// GetVirtualNetworks retrieves all Virtual Networks configured on a Managed System.
func (c *RestClient) GetVirtualNetworks(ctx context.Context, sysUUID string) ([]VirtualNetwork, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualNetwork", c.hmcIP, sysUUID)


	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}


	if resp.StatusCode == http.StatusNoContent {
		return []VirtualNetwork{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Networks: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces: %v", err)
	}

	var networks []VirtualNetwork

	// 1. Locate all <entry> tags in the Atom feed
	entries := doc.FindElements("//entry")

	// 2. Loop through each entry to extract the nested VirtualNetwork payload
	for _, entry := range entries {
		vnetElem := entry.FindElement(".//VirtualNetwork")
		if vnetElem == nil {
			continue
		}

		// Create a new document with the VirtualNetwork element as the root
		entryDoc := etree.NewDocument()
		entryDoc.SetRoot(vnetElem.Copy())
		entryBytes, _ := entryDoc.WriteToBytes()

		var vnet VirtualNetwork
		if err := xml.Unmarshal(entryBytes, &vnet); err != nil {
			continue
		}
		networks = append(networks, vnet)
	}


	return networks, nil
}

// GetVirtualNetwork retrieves the detailed configuration of a specific Virtual Network.
func (c *RestClient) GetVirtualNetwork(ctx context.Context, sysUUID, vnetUUID string) (*VirtualNetwork, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualNetwork/%s", c.hmcIP, sysUUID, vnetUUID)


	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Network: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, err
	}

	vnetElem := doc.FindElement("//VirtualNetwork")
	if vnetElem == nil {
		return nil, fmt.Errorf("VirtualNetwork element not found in XML response")
	}

	vnetDoc := etree.NewDocument()
	vnetDoc.SetRoot(vnetElem.Copy())
	vnetBytes, _ := vnetDoc.WriteToBytes()

	var vnet VirtualNetwork
	if err := xml.Unmarshal(vnetBytes, &vnet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VirtualNetwork: %v", err)
	}

	return &vnet, nil
}

// CreateVirtualNetwork provisions a new Virtual Network (VLAN) on the Managed System.
// IBM documentation dictates a PUT method to the base VirtualNetwork endpoint for creation.
func (c *RestClient) CreateVirtualNetwork(ctx context.Context, sysUUID string, req CreateVirtualNetworkRequest) (*VirtualNetwork, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualNetwork", c.hmcIP, sysUUID)


	// The HMC strictly enforces XML element order (AssociatedSwitch -> NetworkName -> NetworkVLANID -> TaggedNetwork)
	payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<VirtualNetwork:VirtualNetwork xmlns:VirtualNetwork="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                               xmlns="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                               schemaVersion="V1_1_0">
    <Metadata><Atom/></Metadata>
    <AssociatedSwitch href="https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s" rel="related"/>
    <NetworkName>%s</NetworkName>
    <NetworkVLANID>%d</NetworkVLANID>
    <TaggedNetwork>%t</TaggedNetwork>
</VirtualNetwork:VirtualNetwork>`, c.hmcIP, sysUUID, req.VSwitchUUID, req.NetworkName, req.NetworkVLANID, req.TaggedNetwork)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Session", c.session)
	httpReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=VirtualNetwork")
	httpReq.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("creation failed (%s): %s", resp.Status, string(body))
	}

	// Parse the response to get the new configuration
	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces: %v", err)
	}

	vnetElem := doc.FindElement("//VirtualNetwork")
	if vnetElem == nil {
		return nil, fmt.Errorf("failed to extract VirtualNetwork from response")
	}

	vnetDoc := etree.NewDocument()
	vnetDoc.SetRoot(vnetElem.Copy())
	vnetBytes, _ := vnetDoc.WriteToBytes()

	var vnet VirtualNetwork
	xml.Unmarshal(vnetBytes, &vnet)


	return &vnet, nil
}

// UpdateVirtualNetwork modifies the properties of an existing Virtual Network.
// Note: Per IBM documentation, only the NetworkName property can be modified.
func (c *RestClient) UpdateVirtualNetwork(ctx context.Context, sysUUID, vnetUUID, newName string) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualNetwork/%s", c.hmcIP, sysUUID, vnetUUID)


	// 1. Pristine GET to preserve all immutable configuration parameters
	getReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	getReq.Header.Set("X-API-Session", c.session)
	getReq.Header.Set("Accept", "application/vnd.ibm.powervm.uom+xml")

	getResp, err := c.client.Do(getReq)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()
	rawXML, _ := io.ReadAll(getResp.Body)

	if getResp.StatusCode != 200 {
		return fmt.Errorf("pre-flight GET failed: %s", string(rawXML))
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawXML); err != nil {
		return fmt.Errorf("failed to parse pristine XML: %v", err)
	}

	vnetElem := doc.FindElement(".//*[local-name()='VirtualNetwork']")
	if vnetElem == nil {
		return fmt.Errorf("VirtualNetwork element not found in pristine XML")
	}

	// 2. Modify the permitted element
	nameElem := vnetElem.FindElement(".//*[local-name()='NetworkName']")
	if nameElem == nil {
		nameElem = vnetElem.CreateElement("NetworkName")
	}
	nameElem.SetText(newName)

	// 3. POST the update back
	postDoc := etree.NewDocument()
	postDoc.SetRoot(vnetElem.Copy())
	postXML, _ := postDoc.WriteToString()

	postReq, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(postXML))
	postReq.Header.Set("X-API-Session", c.session)
	postReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=VirtualNetwork")
	postReq.Header.Set("Accept", "application/atom+xml")


	postResp, err := c.client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	body, err := io.ReadAll(postResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if postResp.StatusCode >= 400 {
		return fmt.Errorf("POST failed (%s): %s", postResp.Status, string(body))
	}


	return nil
}

// DeleteVirtualNetwork removes a Virtual Network from the Managed System.
// Note: Deletion will fail if the Virtual Network is equivalent to the Trunk Adapter PVID,
// or if it is still attached to a NetworkBridge.
func (c *RestClient) DeleteVirtualNetwork(ctx context.Context, sysUUID, vnetUUID string) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualNetwork/%s", c.hmcIP, sysUUID, vnetUUID)


	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %v", err)
	}

	req.Header.Set("X-API-Session", c.session)
	// The HMC returned 406 Not Acceptable for DELETE with
	// Accept: application/vnd.ibm.powervm.uom+xml; use atom+xml.
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete Virtual Network (%s): %s", resp.Status, string(body))
	}


	return nil
}

// GetVirtualSwitch retrieves the detailed XML configuration of a specific Virtual Switch.
func (c *RestClient) GetVirtualSwitch(ctx context.Context, sysUUID, switchUUID string) (*VirtualSwitch, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s", c.hmcIP, sysUUID, switchUUID)


	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Virtual Switch: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, err
	}

	vswitchElem := doc.FindElement("//VirtualSwitch")
	if vswitchElem == nil {
		return nil, fmt.Errorf("VirtualSwitch element not found in XML response")
	}

	vSwitch := VirtualSwitch{}

	// Extract Core properties using etree to match your existing struct
	if atomID := vswitchElem.FindElement(".//AtomID"); atomID != nil {
		vSwitch.UUID = atomID.Text()
	}
	if switchID := vswitchElem.FindElement(".//SwitchID"); switchID != nil {
		vSwitch.SwitchID = switchID.Text()
	}
	if switchName := vswitchElem.FindElement(".//SwitchName"); switchName != nil {
		vSwitch.SwitchName = switchName.Text()
	}
	if switchMode := vswitchElem.FindElement(".//SwitchMode"); switchMode != nil {
		vSwitch.SwitchMode = switchMode.Text()
	}

	// Extract attached VirtualNetwork Links
	for _, link := range vswitchElem.FindElements(".//VirtualNetworks/link") {
		if href := link.SelectAttrValue("href", ""); href != "" {
			vSwitch.VirtualNetworks = append(vSwitch.VirtualNetworks, href)
		}
	}


	return &vSwitch, nil
}

// CreateVirtualSwitch provisions a new Virtual Switch on the Managed System.
func (c *RestClient) CreateVirtualSwitch(ctx context.Context, sysUUID string, req CreateVirtualSwitchRequest) (*VirtualSwitch, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch", c.hmcIP, sysUUID)


	// Default to Veb if not specified
	mode := "Veb"
	if req.SwitchMode != "" {
		mode = req.SwitchMode
	}

	payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<VirtualSwitch:VirtualSwitch xmlns:VirtualSwitch="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                             xmlns="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                             schemaVersion="V1_1_0">
    <Metadata><Atom/></Metadata>
    <SwitchName>%s</SwitchName>
    <SwitchMode>%s</SwitchMode>
</VirtualSwitch:VirtualSwitch>`, req.SwitchName, mode)

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Session", c.session)
	httpReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=VirtualSwitch")
	httpReq.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("creation failed (%s): %s", resp.Status, string(body))
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces: %v", err)
	}

	vswitchElem := doc.FindElement("//VirtualSwitch")
	if vswitchElem == nil {
		return nil, fmt.Errorf("failed to extract VirtualSwitch from response")
	}

	vSwitch := VirtualSwitch{}

	if atomID := vswitchElem.FindElement(".//AtomID"); atomID != nil {
		vSwitch.UUID = atomID.Text()
	}
	if switchID := vswitchElem.FindElement(".//SwitchID"); switchID != nil {
		vSwitch.SwitchID = switchID.Text()
	}
	if switchName := vswitchElem.FindElement(".//SwitchName"); switchName != nil {
		vSwitch.SwitchName = switchName.Text()
	}
	if switchMode := vswitchElem.FindElement(".//SwitchMode"); switchMode != nil {
		vSwitch.SwitchMode = switchMode.Text()
	}


	return &vSwitch, nil
}

// UpdateVirtualSwitch modifies the properties of an existing Virtual Switch.
// According to IBM docs, only SwitchName and SwitchMode are modifiable.
func (c *RestClient) UpdateVirtualSwitch(ctx context.Context, sysUUID, switchUUID, newName, newMode string) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s", c.hmcIP, sysUUID, switchUUID)


	// 1. Pristine GET to preserve all immutable configuration parameters
	getReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	getReq.Header.Set("X-API-Session", c.session)
	getReq.Header.Set("Accept", "application/vnd.ibm.powervm.uom+xml")

	getResp, err := c.client.Do(getReq)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()
	rawXML, _ := io.ReadAll(getResp.Body)

	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("pre-flight GET failed: %s", string(rawXML))
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawXML); err != nil {
		return fmt.Errorf("failed to parse pristine XML: %v", err)
	}

	vswitchElem := doc.FindElement(".//*[local-name()='VirtualSwitch']")
	if vswitchElem == nil {
		return fmt.Errorf("VirtualSwitch element not found in pristine XML")
	}

	// 2. Modify the permitted elements
	if newName != "" {
		nameElem := vswitchElem.FindElement(".//*[local-name()='SwitchName']")
		if nameElem == nil {
			nameElem = vswitchElem.CreateElement("SwitchName")
		}
		nameElem.SetText(newName)
	}

	if newMode != "" {
		modeElem := vswitchElem.FindElement(".//*[local-name()='SwitchMode']")
		if modeElem == nil {
			modeElem = vswitchElem.CreateElement("SwitchMode")
		}
		modeElem.SetText(newMode)
	}

	// 3. POST the update back
	postDoc := etree.NewDocument()
	postDoc.SetRoot(vswitchElem.Copy())
	postXML, _ := postDoc.WriteToString()

	postReq, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(postXML))
	postReq.Header.Set("X-API-Session", c.session)
	postReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=VirtualSwitch")
	postReq.Header.Set("Accept", "application/atom+xml")


	postResp, err := c.client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	body, err := io.ReadAll(postResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if postResp.StatusCode >= 400 {
		return fmt.Errorf("POST failed (%s): %s", postResp.Status, string(body))
	}


	return nil
}

// DeleteVirtualSwitch removes a Virtual Switch from the Managed System.
func (c *RestClient) DeleteVirtualSwitch(ctx context.Context, sysUUID, switchUUID string) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/VirtualSwitch/%s", c.hmcIP, sysUUID, switchUUID)


	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %v", err)
	}

	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/vnd.ibm.powervm.uom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete Virtual Switch (%s): %s", resp.Status, string(body))
	}


	return nil
}


// GetNetworkBridges retrieves all Network Bridges configured on a Managed System.
func (c *RestClient) GetNetworkBridges(ctx context.Context, sysUUID string) ([]NetworkBridge, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/NetworkBridge", c.hmcIP, sysUUID)


	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}


	if resp.StatusCode == http.StatusNoContent {
		return []NetworkBridge{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Network Bridges: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, fmt.Errorf("failed to strip namespaces: %v", err)
	}

	var bridges []NetworkBridge

	// 1. Locate all <entry> tags in the Atom feed
	entries := doc.FindElements("//entry")

	// 2. Loop through each entry to extract the nested NetworkBridge payload
	for _, entry := range entries {
		bridgeElem := entry.FindElement(".//NetworkBridge")
		if bridgeElem == nil {
			continue
		}

		entryDoc := etree.NewDocument()
		entryDoc.SetRoot(bridgeElem.Copy())
		entryBytes, _ := entryDoc.WriteToBytes()

		var bridge NetworkBridge
		if err := xml.Unmarshal(entryBytes, &bridge); err != nil {
			continue
		}
		bridges = append(bridges, bridge)
	}


	return bridges, nil
}

// GetNetworkBridge retrieves the detailed configuration of a specific Network Bridge.
func (c *RestClient) GetNetworkBridge(ctx context.Context, sysUUID, bridgeUUID string) (*NetworkBridge, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/NetworkBridge/%s", c.hmcIP, sysUUID, bridgeUUID)


	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-API-Session", c.session)
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}


	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Network Bridge: %s", resp.Status)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, err
	}

	bridgeElem := doc.FindElement("//NetworkBridge")
	if bridgeElem == nil {
		return nil, fmt.Errorf("NetworkBridge element not found in XML response")
	}

	bridgeDoc := etree.NewDocument()
	bridgeDoc.SetRoot(bridgeElem.Copy())
	bridgeBytes, _ := bridgeDoc.WriteToBytes()

	var bridge NetworkBridge
	if err := xml.Unmarshal(bridgeBytes, &bridge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NetworkBridge: %v", err)
	}

	return &bridge, nil
}


// normalizeNetworkBridgeLoadGroups merges the preferred LoadGroups field and the
// deprecated LoadGroupVLANs field into one list. A bridge without load balancing
// defaults to a single LoadGroup whose PVID equals the bridge PortVLANID. For load
// balancing no group is invented; the caller must supply at least two.
func normalizeNetworkBridgeLoadGroups(req CreateNetworkBridgeRequest) ([]NetworkBridgeLoadGroup, error) {
	if len(req.LoadGroups) > 0 && len(req.LoadGroupVLANs) > 0 {
		return nil, fmt.Errorf("specify either LoadGroups or LoadGroupVLANs, not both")
	}

	groups := append([]NetworkBridgeLoadGroup(nil), req.LoadGroups...)
	for _, vlan := range req.LoadGroupVLANs {
		groups = append(groups, NetworkBridgeLoadGroup{PortVLANID: vlan})
	}

	if len(groups) == 0 && !req.LoadBalancingEnabled {
		groups = []NetworkBridgeLoadGroup{{PortVLANID: req.PortVLANID}}
	}
	return groups, nil
}

// validateCreateNetworkBridgeRequest applies the NetworkBridge rules documented in the
// HMC REST API: failover needs two SEAs, at least one LoadGroup (two with load
// balancing), and one LoadGroup PVID must equal the bridge PortVLANID. ControlChannelID
// is not enforced because it is optional on ManagementVLANForControlChannelCapable
// systems; the HMC reports the error when it is required and missing.
func validateCreateNetworkBridgeRequest(req CreateNetworkBridgeRequest, groups []NetworkBridgeLoadGroup) error {
	if req.PortVLANID < 1 || req.PortVLANID > 4094 {
		return fmt.Errorf("PortVLANID must be between 1 and 4094")
	}
	if req.PrimaryViosUUID == "" || req.PrimaryBackingDevice == "" {
		return fmt.Errorf("PrimaryViosUUID and PrimaryBackingDevice are required")
	}
	if req.ControlChannelID < 0 || req.ControlChannelID > 4094 {
		return fmt.Errorf("ControlChannelID must be between 1 and 4094 when specified")
	}

	if req.FailoverEnabled {
		if req.SecondaryViosUUID == "" || req.SecondaryBackingDevice == "" {
			return fmt.Errorf("SecondaryViosUUID and SecondaryBackingDevice are required when FailoverEnabled is true")
		}
		if req.PrimaryViosUUID == req.SecondaryViosUUID {
			return fmt.Errorf("primary and secondary VIOS must be different")
		}
	}

	minimumGroups := 1
	if req.LoadBalancingEnabled {
		if !req.FailoverEnabled {
			return fmt.Errorf("LoadBalancingEnabled requires FailoverEnabled")
		}
		minimumGroups = 2
	}
	if len(groups) < minimumGroups {
		return fmt.Errorf("NetworkBridge requires at least %d LoadGroup(s) (set LoadGroups or LoadGroupVLANs); "+
			"load balancing needs two or more LoadGroups, not necessarily newly created Virtual Networks", minimumGroups)
	}

	seen := make(map[int]struct{}, len(groups))
	bridgePVIDPresent := false
	for i, g := range groups {
		if g.PortVLANID < 1 || g.PortVLANID > 4094 {
			return fmt.Errorf("LoadGroups[%d].PortVLANID %d must be between 1 and 4094", i, g.PortVLANID)
		}
		if _, dup := seen[g.PortVLANID]; dup {
			return fmt.Errorf("duplicate LoadGroup PVID %d", g.PortVLANID)
		}
		seen[g.PortVLANID] = struct{}{}
		if g.PortVLANID == req.PortVLANID {
			bridgePVIDPresent = true
		}
	}
	if !bridgePVIDPresent {
		return fmt.Errorf("one LoadGroup must use PortVLANID %d, matching the NetworkBridge PortVLANID", req.PortVLANID)
	}
	return nil
}

// truncateErrorBody trims an HMC response body for inclusion in an error message.
func truncateErrorBody(body []byte, limit int) string {
	text := strings.TrimSpace(string(body))
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

// CreateNetworkBridge provisions a new Network Bridge (Shared Ethernet Adapter wrapper).
// It automatically supports both standard Active/Standby HA Failover and Active/Active Load Balancing layouts.
func (c *RestClient) CreateNetworkBridge(ctx context.Context, sysUUID string, req CreateNetworkBridgeRequest) (*NetworkBridge, error) {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/NetworkBridge", c.hmcIP, sysUUID)

	// 1. Normalize and validate the LoadGroups before contacting the HMC
	groups, err := normalizeNetworkBridgeLoadGroups(req)
	if err != nil {
		return nil, err
	}
	if err := validateCreateNetworkBridgeRequest(req, groups); err != nil {
		return nil, err
	}

	// 2. Resolve Virtual Networks for groups that did not supply explicit UUIDs
	var vNets []VirtualNetwork
	for _, g := range groups {
		if len(g.VirtualNetworkUUIDs) == 0 {
			vNets, err = c.GetVirtualNetworks(ctx, sysUUID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch Virtual Networks for infrastructure alignment: %v", err)
			}
			break
		}
	}

	// 3. Dynamically assemble the ordered sequential LoadGroups child elements
	var loadGroupsSB strings.Builder
	loadGroupsSB.WriteString("<LoadGroups schemaVersion=\"V1_0\">")

	for _, g := range groups {
		uuids := g.VirtualNetworkUUIDs
		if len(uuids) == 0 {
			for _, vn := range vNets {
				if vn.NetworkVLANID == g.PortVLANID {
					uuids = []string{vn.UUID}
					break
				}
			}
		}
		if len(uuids) == 0 {
			return nil, fmt.Errorf("failed to find an existing Virtual Network matching VLAN ID %d; please create it first", g.PortVLANID)
		}

		loadGroupsSB.WriteString(fmt.Sprintf(`
        <LoadGroup schemaVersion="V1_0">
            <PortVLANID>%d</PortVLANID>
            <VirtualNetworks>`, g.PortVLANID))
		for _, u := range uuids {
			loadGroupsSB.WriteString(fmt.Sprintf(`
                <link href="https://%s:443/rest/api/uom/ManagedSystem/%s/VirtualNetwork/%s" rel="related"/>`, c.hmcIP, sysUUID, u))
		}
		loadGroupsSB.WriteString(`
            </VirtualNetworks>
        </LoadGroup>`)
	}
	loadGroupsSB.WriteString("\n    </LoadGroups>")
	loadGroupsXML := loadGroupsSB.String()

	// 4. Construct optional Failover HA payload blocks (Strict JAXB Sorting Rules)
	controlChannelXML := ""
	secondarySEAXML := ""
	if req.ControlChannelID > 0 {
		controlChannelXML = fmt.Sprintf("<ControlChannelID>%d</ControlChannelID>", req.ControlChannelID)
	}
	if req.FailoverEnabled {
		// Order sequence: AssignedVirtualIOServer -> BackingDeviceChoice -> JumboFramesEnabled -> IsPrimary -> LargeSend
		secondarySEAXML = fmt.Sprintf(`
        <SharedEthernetAdapter schemaVersion="V1_0">
            <AssignedVirtualIOServer href="https://%s:443/rest/api/uom/VirtualIOServer/%s" rel="related"/>
            <BackingDeviceChoice>
                <EthernetBackingDevice schemaVersion="V1_0">
                    <DeviceName>%s</DeviceName>
                </EthernetBackingDevice>
            </BackingDeviceChoice>
            <JumboFramesEnabled>%t</JumboFramesEnabled>
            <IsPrimary>false</IsPrimary>
            <LargeSend>%t</LargeSend>
        </SharedEthernetAdapter>`, c.hmcIP, req.SecondaryViosUUID, req.SecondaryBackingDevice, req.JumboFramesEnabled, req.LargeSend)
	}

	// 5. Construct Final Structural XML Representation Payload (Root Alphabetical Sequence)
	payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<NetworkBridge:NetworkBridge xmlns:NetworkBridge="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                             xmlns="http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/"
                             schemaVersion="V1_1_0">
    <Metadata><Atom/></Metadata>
    %s
    <FailoverEnabled>%t</FailoverEnabled>
    <LoadBalancingEnabled>%t</LoadBalancingEnabled>
    %s
    <PortVLANID>%d</PortVLANID>
    <SharedEthernetAdapters schemaVersion="V1_0">
        <SharedEthernetAdapter schemaVersion="V1_0">
            <AssignedVirtualIOServer href="https://%s:443/rest/api/uom/VirtualIOServer/%s" rel="related"/>
            <BackingDeviceChoice>
                <EthernetBackingDevice schemaVersion="V1_0">
                    <DeviceName>%s</DeviceName>
                </EthernetBackingDevice>
            </BackingDeviceChoice>
            <JumboFramesEnabled>%t</JumboFramesEnabled>
            <IsPrimary>true</IsPrimary>
            <LargeSend>%t</LargeSend>
        </SharedEthernetAdapter>
        %s
    </SharedEthernetAdapters>
</NetworkBridge:NetworkBridge>`,
		controlChannelXML, req.FailoverEnabled, req.LoadBalancingEnabled,
		loadGroupsXML,
		req.PortVLANID,
		c.hmcIP, req.PrimaryViosUUID, req.PrimaryBackingDevice, req.JumboFramesEnabled, req.LargeSend,
		secondarySEAXML)

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-API-Session", c.session)
	httpReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=NetworkBridge")
	httpReq.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		message := truncateErrorBody(body, 2000)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("NetworkBridge creation failed: %s: %s", resp.Status, message)
	}

	doc, err := xmlStripNamespace(body)
	if err != nil {
		return nil, err
	}

	bridgeElem := doc.FindElement("//NetworkBridge")
	if bridgeElem == nil {
		return nil, fmt.Errorf("failed to extract NetworkBridge structural entity from response payload")
	}

	bridgeDoc := etree.NewDocument()
	bridgeDoc.SetRoot(bridgeElem.Copy())
	bridgeBytes, _ := bridgeDoc.WriteToBytes()

	var bridge NetworkBridge
	if err := xml.Unmarshal(bridgeBytes, &bridge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provisioned configuration profile mappings: %v", err)
	}

	return &bridge, nil
}

// UpdateNetworkBridge modifies properties of an active asset utilizing a pristine GET-POST workflow transaction pattern.
func (c *RestClient) UpdateNetworkBridge(ctx context.Context, sysUUID, bridgeUUID string, failoverEnabled, loadBalancingEnabled, largeSend, jumboFrames bool) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/NetworkBridge/%s", c.hmcIP, sysUUID, bridgeUUID)


	// 1. Fetch current layout definition state
	getReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	getReq.Header.Set("X-API-Session", c.session)
	getReq.Header.Set("Accept", "application/vnd.ibm.powervm.uom+xml")

	getResp, err := c.client.Do(getReq)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()
	rawXML, _ := io.ReadAll(getResp.Body)

	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("pre-flight GET configuration tree inquiry failed: %s", string(rawXML))
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawXML); err != nil {
		return fmt.Errorf("failed to digest active response mapping tree configurations: %v", err)
	}

	bridgeElem := doc.FindElement(".//*[local-name()='NetworkBridge']")
	if bridgeElem == nil {
		return fmt.Errorf("NetworkBridge context tag node tracking reference could not be recovered")
	}

	// 2. Adjust root scalar settings
	if foElem := bridgeElem.FindElement(".//*[local-name()='FailoverEnabled']"); foElem != nil {
		foElem.SetText(fmt.Sprintf("%t", failoverEnabled))
	}
	if lbElem := bridgeElem.FindElement(".//*[local-name()='LoadBalancingEnabled']"); lbElem != nil {
		lbElem.SetText(fmt.Sprintf("%t", loadBalancingEnabled))
	}

	// 3. Update nested loops elements
	seas := bridgeElem.FindElements(".//*[local-name()='SharedEthernetAdapter']")
	for _, sea := range seas {
		if lsElem := sea.FindElement(".//*[local-name()='LargeSend']"); lsElem != nil {
			lsElem.SetText(fmt.Sprintf("%t", largeSend))
		} else {
			sea.CreateElement("LargeSend").SetText(fmt.Sprintf("%t", largeSend))
		}

		if jfElem := sea.FindElement(".//*[local-name()='JumboFramesEnabled']"); jfElem != nil {
			jfElem.SetText(fmt.Sprintf("%t", jumboFrames))
		} else {
			sea.CreateElement("JumboFramesEnabled").SetText(fmt.Sprintf("%t", jumboFrames))
		}
	}

	// 4. Dispatch the updated data tree
	postDoc := etree.NewDocument()
	postDoc.SetRoot(bridgeElem.Copy())
	postXML, _ := postDoc.WriteToString()

	postReq, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(postXML))
	postReq.Header.Set("X-API-Session", c.session)
	postReq.Header.Set("Content-Type", "application/vnd.ibm.powervm.uom+xml; type=NetworkBridge")
	postReq.Header.Set("Accept", "application/atom+xml")


	postResp, err := c.client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	body, err := io.ReadAll(postResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if postResp.StatusCode >= 400 {
		return fmt.Errorf("POST layout specification modifications rejected (%s): %s", postResp.Status, string(body))
	}

	return nil
}
// DeleteNetworkBridge removes a Network Bridge from the Managed System.
// This automatically breaks down the underlying SEA on the VIOS.
func (c *RestClient) DeleteNetworkBridge(ctx context.Context, sysUUID, bridgeUUID string) error {
	url := fmt.Sprintf("https://%s/rest/api/uom/ManagedSystem/%s/NetworkBridge/%s", c.hmcIP, sysUUID, bridgeUUID)


	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %v", err)
	}

	req.Header.Set("X-API-Session", c.session)
	// The HMC returned 406 Not Acceptable for DELETE with
	// Accept: application/vnd.ibm.powervm.uom+xml; use atom+xml like the
	// NetworkBridge POST and other DELETE calls in this package.
	req.Header.Set("Accept", "application/atom+xml")


	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete Network Bridge (%s): %s", resp.Status, string(body))
	}


	return nil
}

