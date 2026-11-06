package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	hmc "github.com/IBM/infra-go-sdk/phmc"
	exutil "github.com/IBM/infra-go-sdk/phmc/examples/exutil"
)

func main() {
	// NOTE:
	// This example calls GetNetworkBootDevices for a VIOS using the name-based
	// job parameters (managedSystemName, laparName, profileName, loggedinuser).
	// The VIOS must be powered off; use the getviosnetbootdeviceimm example to
	// have the HMC shut a running VIOS down immediately.
	// =========================================================================
	// CONFIGURATION & FLAGS
	// =========================================================================
	hmcIP := flag.String("hmc-ip", "", "HMC IP address")
	username := flag.String("hmc-user", "", "HMC username")
	password := flag.String("hmc-pass", "", "HMC password")
	sysName := flag.String("system-name", "", "Managed System Name")
	viosName := flag.String("vios-name", "", "Target VIOS Name")
	profileName := flag.String("profile-name", "", "VIOS Profile Name (leave blank to auto-detect)")
	debug     := flag.Bool("debug",      false, "Log each HTTP request/response (bodies truncated at 2048 bytes)")
	debugFull := flag.Bool("debug-full",  false, "Log each HTTP request/response with full body (no truncation)")
	flag.Parse()

	if *hmcIP == "" || *username == "" || *password == "" || *sysName == "" || *viosName == "" {
		log.Fatal("❌ Error: hmc-ip, hmc-user, hmc-pass, system-name, and vios-name are required")
	}

	fmt.Println("=================================================================")
	fmt.Println("  VIOS Network Boot Device Information Utility")
	fmt.Println("=================================================================")
	fmt.Printf("HMC:    %s\n", *hmcIP)
	fmt.Printf("System: %s\n", *sysName)
	fmt.Printf("VIOS:   %s\n", *viosName)
	fmt.Println("=================================================================")

	// 1. Initialize & Login
	fmt.Println("\nStep 1: Connecting to HMC...")
	client := exutil.NewClient(*hmcIP, *debug, *debugFull)
	if err := client.Login(context.Background(), *username, *password); err != nil {
		log.Fatalf("❌ Login failed: %v", err)
	}
	defer client.Logoff(context.Background())
	fmt.Println("✅ Connected to HMC")

	// 2. Resolve System UUID
	fmt.Printf("Step 2: Resolving System UUID for '%s'...\n", *sysName)
	_, sysUUID, err := client.GetManagedSystemByNameQuick(context.Background(), *sysName)
	if err != nil || sysUUID == "" {
		log.Fatalf("❌ Managed System '%s' not found: %v", *sysName, err)
	}
	fmt.Printf("✅ System UUID: %s\n", sysUUID)

	// 3. Resolve VIOS UUID
	fmt.Printf("Step 3: Resolving VIOS UUID for '%s'...\n", *viosName)
	viosUUID, err := hmc.GetViosID(context.Background(), client, sysUUID, *viosName)
	if err != nil || viosUUID == "" {
		log.Fatalf("❌ VIOS '%s' not found on system '%s': %v", *viosName, *sysName, err)
	}
	fmt.Printf("✅ VIOS UUID: %s\n", viosUUID)

	// 4. Check power state and resolve the profile name
	fmt.Println("Step 4: Fetching VIOS details...")
	viosDetailed, err := client.GetVirtualIOServer(context.Background(), viosUUID)
	if err != nil {
		log.Fatalf("❌ Failed to get VIOS details: %v", err)
	}
	if viosDetailed.PartitionState == "running" {
		fmt.Println("\n⚠️  WARNING: VIOS is currently RUNNING")
		fmt.Println("   The GetNetworkBootDevices job requires the VIOS to be powered off first.")
		fmt.Println("   Power it off, or use the getviosnetbootdeviceimm example. Exiting.")
		return
	}

	actualProfileName := *profileName
	if actualProfileName == "" {
		viosList, err := client.GetVirtualIOServersQuick(context.Background(), sysUUID)
		if err != nil {
			log.Fatalf("❌ Failed to fetch VIOS list from system: %v", err)
		}
		for _, v := range viosList {
			if v.UUID == viosUUID {
				actualProfileName = v.LastActivatedProfile
				break
			}
		}
	}
	if actualProfileName == "" {
		log.Fatal("❌ Could not determine profile name; pass -profile-name")
	}
	fmt.Printf("✅ Target Profile: %s\n", actualProfileName)

	// 5. Fetch Network Boot Devices
	fmt.Println("\nStep 5: Firing GetNetworkBootDevices Job for VIOS...")
	bootDevices, err := client.GetNetworkBootDevicesForVios(context.Background(), viosUUID, *sysName, *viosName, actualProfileName, *username)
	if err != nil {
		log.Fatalf("❌ GetNetworkBootDevices failed: %v", err)
	}

	if len(bootDevices) == 0 {
		fmt.Printf("⚠️  No network boot devices found in profile for VIOS '%s'\n", *viosName)
		return
	}

	fmt.Printf("✅ Retrieved %d network boot device(s) from profile\n", len(bootDevices))

	// 6. Display Network Boot Device Information
	fmt.Println("\n=================================================================")
	fmt.Println("  VIOS NETWORK BOOT DEVICE INFORMATION")
	fmt.Println("=================================================================")

	for i, device := range bootDevices {
		fmt.Printf("--- Boot Device %d ---\n", i+1)
		fmt.Printf("Device Name:         %s\n", device.DeviceName)
		fmt.Printf("Device Type:         %s\n", device.DeviceType)
		fmt.Printf("Location Code:       %s\n", device.LocationCode)
		fmt.Printf("MAC Address:         %s\n", device.MACAddress)
		fmt.Println(strings.Repeat("-", 65))
	}
}
