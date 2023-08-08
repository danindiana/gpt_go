package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/inancgumus/screen"
	"github.com/shirou/gopsutil/mem"
)


func main() {
	screen.Clear()

	for {
	    screen.MoveTopLeft()

	    // Display RAM utilization
	    v, err := mem.VirtualMemory()
	    if err != nil {
	        log.Fatalf("Failed to get RAM utilization: %v", err)
	    }
	    fmt.Printf("RAM Utilization: %s %.2f%%\n\n", generateBar(v.UsedPercent), v.UsedPercent)

	    // Display disk drive utilization
	    fmt.Println("Disk Drive Utilization")
	    fmt.Println("----------------------")

	    driveNames, err := getDriveNames()
	    if err != nil {
	        log.Fatal(err)
	    }

	    diskUsage := generateRandomDiskUsage(len(driveNames))
	    for i, usage := range diskUsage {
	        driveLabel := driveNames[i]
	        bar := generateBar(usage)
	        fmt.Printf("%s: %s %.2f%%\n", driveLabel, bar, usage)
	    }

	    // Introduce a delay
	    time.Sleep(2 * time.Second)
	}
}

func generateBar(percentage float64) string {
	const totalBars = 50
	barsToShow := int((percentage / 100) * totalBars)
	return strings.Repeat("█", barsToShow) + strings.Repeat(" ", totalBars-barsToShow)
}

func generateRandomDiskUsage(count int) []float64 {
	rand.Seed(time.Now().UnixNano())
	diskUsage := make([]float64, count)
	for i := range diskUsage {
		diskUsage[i] = rand.Float64() * 100
	}
	return diskUsage
}

func getDriveNames() ([]string, error) {
	var driveNames []string

	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return nil, err
	}
	getLogicalDriveStrings, err := kernel32.FindProc("GetLogicalDriveStringsW")
	if err != nil {
		return nil, err
	}

	buffer := make([]uint16, 254)
	ret, _, err := getLogicalDriveStrings.Call(uintptr(len(buffer)*2), uintptr(unsafe.Pointer(&buffer[0])))
	if ret == 0 {
		return nil, err
	}

	ucs := utf16.Decode(buffer)
	driveString := string(ucs)
	for _, drive := range strings.Split(driveString, "\x00") {
		if drive != "" {
			driveNames = append(driveNames, drive)
		}
	}

	return driveNames, nil
}


