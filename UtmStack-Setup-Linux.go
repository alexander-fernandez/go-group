package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// To handle the license structure
type licenseDataJSON struct {
	ID                int
	IrderID           int
	ProductID         int
	UserID            int
	LicenseKey        string
	ExpiresAt         string
	ValidFor          int
	Source            int
	Status            int
	TimesActivated    int
	TimesActivatedMax int
	CreatedAt         string
	CreatedBy         int
	UpdatedAt         string
	UpdatedBy         int
}

// To handle the license structure
type licenseJSON struct {
	Success bool
	Code    string
	Message string
	Data    licenseDataJSON
}

// To be used by the progress bar
type myBar struct {
	percent int64  // progress percentage
	cur     int64  // current progress
	total   int64  // total value for progress
	rate    string // the actual progress bar to be printed
	graph   string // the fill value for progress bar
}

// Global variables
var adminuser, adminpass, domain, prefix, licAct, licExp, licName, licMail, licKey, version string
var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
var licensed bool

func main() {
	// Checking if user is root
	if os.Geteuid() != 0 {
		fmt.Println("[ERROR] => [UTMSTACK]: This Console installer must be run as ROOT user. Exiting ...")
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: ROOT user ... Ok.")
	}

	// TEST if APT is Busy
	// lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'
	outPID, _ := exec.Command("sh", "-c", `lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'`).Output()
	strPID := string(outPID)
	trimPID := strings.TrimRight(strPID, "\n")

	if trimPID != "" {
		fmt.Println("[ERROR] => [UTMSTACK]: APT-GET Services are busy, PID:", trimPID)
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: APT-GET Services ... Ok.")
	}

	// Section to select
	version = selectInstallType()
	if version == "u" {
		fmt.Println("# [UTMSTACK]: Uninstalling UTMStack Deployment ...")
		utmUninstall()
		fmt.Println("# [UTMSTACK]: Uninstaller is Done ... Ok.")
		os.Exit(0)
	}

	choice := selectProduct()

	// Create elasticsearch user if user 1000 doesn't exists
	outUser, _ := exec.Command("getent", "passwd", "1000").Output()
	strUser := string(outUser)
	trimUser := strings.TrimRight(strUser, "\n")

	if trimUser == "" {
		err := exec.Command("useradd", "-u", "1000", "elasticsearch").Run()
		if err != nil {
			fmt.Println("[ERROR]: Creating user elasticsearch =>", err)
		}
	}

	// mkdir /etc/utmstack/
	if err := os.Mkdir("/etc/utmstack", 0755); err != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was a problem creating: /etc/utmstack/", "| Error:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Folder: /etc/utmstack Created ... Ok.")
	}

	// Copy End User License Agreement (EULA)| Legal and Privacy Information
	copyLicense()

	// Menu tu accept EULA
	agreeEULA()

	// Adding Ubuntu repositories

	// echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs) main restricted universe multiverse" > /etc/apt/sources.list
	fmt.Println("# [UTMSTACK]: Setting up the Ubuntu repository, branch UBUNTU ...")
	cmd := exec.Command("sh", "-c", `echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs) main restricted universe multiverse" > /etc/apt/sources.list`)
	cmd.Run()

	// echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-updates main restricted universe multiverse" >> /etc/apt/sources.list
	fmt.Println("# [UTMSTACK]: Setting up the Ubuntu repository, branch UBUNTU-UPDATES ...")
	cmd = exec.Command("sh", "-c", `echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-updates main restricted universe multiverse" >> /etc/apt/sources.list`)
	cmd.Run()

	// echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-security main restricted universe multiverse" >> /etc/apt/sources.list
	fmt.Println("# [UTMSTACK]: Setting up the Ubuntu repository, branch UBUNTU-SECURITY ...")
	cmd = exec.Command("sh", "-c", `echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-security main restricted universe multiverse" >> /etc/apt/sources.list`)
	cmd.Run()

	// echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-backports main restricted universe multiverse" >> /etc/apt/sources.list
	fmt.Println("# [UTMSTACK]: Setting up the Ubuntu repository, branch UBUNTU-BACKPORTS ...")
	cmd = exec.Command("sh", "-c", `echo "deb http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-backports main restricted universe multiverse" >> /etc/apt/sources.list`)
	cmd.Run()

	// Updating repository index
	fmt.Println("# [UTMSTACK]: Updating repository index ...")
	cmd = exec.Command("apt", "-y", "update")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// TEST if APT is Busy
	// lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'
	outPID, _ = exec.Command("sh", "-c", `lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'`).Output()
	strPID = string(outPID)
	trimPID = strings.TrimRight(strPID, "\n")

	if trimPID != "" {
		fmt.Println("[ERROR] => [UTMSTACK]: APT-GET Services are busy, PID:", trimPID)
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: APT-GET Services ... Ok.")
	}

	// Install curl
	fmt.Println("# [UTMSTACK]: Installing: curl ...")
	cmd = exec.Command("apt", "-y", "install", "curl")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Testing Access to FrontEnd, BackEnd (war) and Docker Registry
	testAccess()

	// Create configuration files (application-prod.yml, compose.yml, config.json, utm_client.sql)
	copyFiles()

	if choice == "p" {
		fmt.Println("Installing Proxy for Data Collection (Just Probe).")
		proxyProbe()
		fmt.Println("Data Collection Proxy Installation is Done ... Ok.")
		os.Exit(0)
	}

	if choice == "m" {
		fmt.Println("Installing Master and Infrastructure ...")
		inputData()
		genCert()
		utmInstall()
		fmt.Println("Master and Infrastructure Installer is Done ... Ok.")
		os.Exit(0)
	}
}

func utmInstall() { // Installer
	// free --mega | awk 'NR==2{printf "%.i", $2}'
	outRAM, _ := exec.Command("sh", "-c", `free --mega | awk 'NR==2{printf "%.i", $2}'`).Output()
	strRAM := string(outRAM)
	trimRAM := strings.TrimRight(strRAM, "\n")

	// lscpu | grep -i 'CPU(s):' | awk '{printf "%.i", $2}'
	outCPU, _ := exec.Command("sh", "-c", `lscpu | grep -i 'CPU(s):' | awk '{printf "%.i", $2}'`).Output()
	strCPU := string(outCPU)
	trimCPU := strings.TrimRight(strCPU, "\n")

	// Selecting Server Size
	intRAM, _ := strconv.Atoi(trimRAM)
	intCPU, _ := strconv.Atoi(trimCPU)
	max := 0
	for {
		fmt.Println("# [UTMSTACK]: Your resources, RAM:", trimRAM, "MB, CPU:", trimCPU, "UNITS")
		fmt.Println("# [UTMSTACK]: According to this, you can install:")
		max = 1
		fmt.Println("(1) Undersized Server deployment")
		if intRAM >= 16384 && intCPU >= 4 {
			fmt.Println("(2) Tiny Server deployment (at least RAM:16Gb, CPU:4 units)")
			max = 2
		}
		if intRAM >= 32768 && intCPU >= 8 {
			fmt.Println("(3) Small Server deployment (at least RAM:32Gb, CPU:8 units)")
			max = 3
		}
		if intRAM >= 65536 && intCPU >= 12 {
			fmt.Println("(4) Medium Server deployment (at least RAM:64Gb, CPU:12 units)")
			max = 4
		}

		fmt.Println("(Q) Quit the installation")
		fmt.Print("Your answer is?: ")
		size := ""
		fmt.Scanln(&size)
		opt := string(size)
		if opt == "0" {
			fmt.Println("# [UTMSTACK]: Exiting the installation ...")
			os.Exit(1)
		}

		if opt == "1" {
			intSize, _ := strconv.Atoi(size)
			if intSize <= max {
				err := exec.Command("sed", "-i", "s|JVM_CORD|2048|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MIN|512|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MAX|1024|g", "/etc/utmstack/compose.yml").Run()
				if err != nil {
					fmt.Println("[ERROR]:", err)
				}
				break
			} else {
				fmt.Println("[ERROR] => [UTMSTACK]: You CAN'T use this option.")
				continue
			}
		}

		if opt == "2" {
			intSize, _ := strconv.Atoi(size)
			if intSize <= max {
				err := exec.Command("sed", "-i", "s|JVM_CORD|8192|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MIN|512|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MAX|1024|g", "/etc/utmstack/compose.yml").Run()
				if err != nil {
					fmt.Println("[ERROR]:", err)
				}
				break
			} else {
				fmt.Println("[ERROR] => [UTMSTACK]: You CAN'T use this option.")
				continue
			}
		}

		if opt == "3" {
			intSize, _ := strconv.Atoi(size)
			if intSize <= max {
				err := exec.Command("sed", "-i", "s|JVM_CORD|16384|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MIN|1024|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MAX|2048|g", "/etc/utmstack/compose.yml").Run()
				if err != nil {
					fmt.Println("[ERROR]:", err)
				}
				break
			} else {
				fmt.Println("[ERROR] => [UTMSTACK]: You CAN'T use this option.")
				continue
			}
		}

		if opt == "4" {
			intSize, _ := strconv.Atoi(size)
			if intSize <= max {
				err := exec.Command("sed", "-i", "s|JVM_CORD|32768|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MIN|1024|g", "/etc/utmstack/compose.yml").Run()
				err = exec.Command("sed", "-i", "s|JAVA_MAX|2048|g", "/etc/utmstack/compose.yml").Run()
				if err != nil {
					fmt.Println("[ERROR]:", err)
				}
				break
			} else {
				fmt.Println("[ERROR] => [UTMSTACK]: You CAN'T use this option.")
				continue
			}
		}

		if opt == "q" || opt == "Q" {
			fmt.Println("# [UTMSTACK]: Exiting the installation ...")
			os.Exit(1)
		}
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}

	storage := []string{
		"/utmstack/data",
		"/utmstack/repo",
	}

	// Checking if "/utmstack/XXXX" exists
	for _, s := range storage {
		if _, err := os.Stat(s); os.IsNotExist(err) {
			if err := os.MkdirAll(s, 0755); err != nil {
				fmt.Println("[ERROR] => [UTMSTACK]: There was a problem creating:", s, "| Error:", err)
			} else {
				fmt.Println("# [UTMSTACK]: Folder:", s, "Created ... Ok.")
			}
		} else {
			fmt.Println("# [UTMSTACK]: Checking:", s, "... Ok.")
		}
	}

	if err := os.Mkdir("/utmstack/data/single", 0755); err != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was a problem creating: '/utmstack/data/single' | Error:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Folder: '/utmstack/data/single' Created ... Ok.")
	}

	if err := os.Mkdir("/utmstack/repo/utm-geoip", 0755); err != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was a problem creating: '/utmstack/repo/utm-geoip' | Error:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Folder: '/utmstack/repo/utm-geoip' Created ... Ok.")
	}

	// Download geoip.zip
	// wget -O /opt/geoip.zip --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/geoip.zip
	fmt.Println("# [UTMSTACK]: Downloading geoip.zip... (~211.2MB, it takes some minutes depending on your bandwidth)")
	strCMD := "wget -O /opt/geoip.zip --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/geoip.zip"
	cmd := exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Checking and copying geoip.zip
	// unzip /opt/geoip.zip
	// cp -R /opt/utm-geoip/* /utmstack/repo/utm-geoip
	// rm -Rf /opt/utm-geoip/
	if _, err := os.Stat("/opt/geoip.zip"); os.IsNotExist(err) {
		fmt.Println("[ERROR] => [UTMSTACK]: Checking: /opt/geoip.zip failed, unable to extract it")
	} else {
		err = exec.Command("unzip", "/opt/geoip.zip", "-d", "/opt").Run()
		strCMD := "cp -R /opt/utm-geoip/* /utmstack/repo/utm-geoip"
		err = exec.Command("/bin/bash", "-c", strCMD).Run()
		err = exec.Command("rm", "-Rf", "/opt/utm-geoip/").Run()
		fmt.Println("# [UTMSTACK]: Extracting geoip.zip ... Ok.")
	}

	filepath.Walk("/utmstack", func(path string, f os.FileInfo, err error) error {
		if e := os.Chown(path, 1000, 1000); e != nil {
			fmt.Println("[ERROR] => [UTMSTACK]: There was a problem setting owner to:", path, "| Error:", e)
		} else {
			fmt.Println("# [UTMSTACK]: Setting owner to:", path, "... Ok.")
		}
		return nil
	})

	// TEST if APT is Busy
	// lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'
	outPID, _ := exec.Command("sh", "-c", `lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'`).Output()
	strPID := string(outPID)
	trimPID := strings.TrimRight(strPID, "\n")

	if trimPID != "" {
		fmt.Println("[ERROR] => [UTMSTACK]: APT-GET Services are busy, PID:", trimPID)
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: APT-GET Services ... Ok.")
	}

	// Installing software-properties-common
	fmt.Println("# [UTMSTACK]: Installing: software-properties-common ...")
	cmd = exec.Command("apt", "-y", "install", "software-properties-common")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// wget --no-check-certificate --quiet https://download.docker.com/linux/ubuntu/gpg -O - | apt-key add -
	strCMD = "wget --no-check-certificate --quiet https://download.docker.com/linux/ubuntu/gpg -O - | apt-key add -"
	err := exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]: Adding Docker's official GPG key to apt =>", err)
	} else {
		fmt.Println("# [UTMSTACK]: Add Docker's official GPG key to apt ... Ok.")
	}

	// add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
	fmt.Println("# [UTMSTACK]: Setting up the stable repository for Docker ... Ok.")
	cmd = exec.Command("sh", "-c", `add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"`)
	cmd.Run()

	// Updating repository index with Docker included
	fmt.Println("# [UTMSTACK]: Updating repository index ...")
	cmd = exec.Command("apt", "-y", "update")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// TEST if APT is Busy
	// lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'
	outPID, _ = exec.Command("sh", "-c", `lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'`).Output()
	strPID = string(outPID)
	trimPID = strings.TrimRight(strPID, "\n")

	if trimPID != "" {
		fmt.Println("[ERROR] => [UTMSTACK]: APT-GET Services are busy, PID:", trimPID)
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: APT-GET Services ... Ok.")
	}

	// Installing apt-transport-https ca-certificates curl gnupg-agent software-properties-common zip unzip apache2-utils
	fmt.Println("# [UTMSTACK]: Installing: apt-transport-https ca-certificates gnupg-agent software-properties-common zip unzip apache2-utils ...")
	cmd = exec.Command("apt", "-y", "install", "apt-transport-https", "ca-certificates", "gnupg-agent", "software-properties-common", "zip", "unzip", "apache2-utils")
	cmd.Stdout = os.Stdout
	cmd.Run()

	//htpasswd -nb $adminuser $adminpass
	// out, _ := exec.Command("htpasswd", "-nb", adminuser, adminpass).Output()
	// str := string(out)
	// trim := strings.TrimRight(str, "\n")
	//traefikPass := strings.Replace(trim, "$", "$$", -1)

	// sed -i "s|utmstack_user|$adminuser|g" compose.yml application-prod.yml
	patternUser := "s|utmstack_user|" + adminuser + "|g"
	err = exec.Command("sed", "-i", patternUser, "/etc/utmstack/compose.yml", "/etc/utmstack/application-prod.yml", "/etc/utmstack/config.json").Run()
	if err != nil {
		fmt.Println("[ERROR]: Setting utmstack_user =>", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting Admin username ... Ok.")
	}

	// sed -i "s|utmstack_pass|$adminpass|g" compose.yml application-prod.yml
	patternPass := "s|utmstack_pass|" + adminpass + "|g"
	err = exec.Command("sed", "-i", patternPass, "/etc/utmstack/compose.yml", "/etc/utmstack/application-prod.yml", "/etc/utmstack/config.json").Run()
	if err != nil {
		fmt.Println("[ERROR]: Setting utmstack_pass =>", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting Admin password ... Ok.")
	}

	// sed -i "s|utmclient.utmstack.com|$domain|g" compose.yml application-prod.yml
	patternDomain := "s|utmclient.utmstack.com|" + domain + "|g"
	err = exec.Command("sed", "-i", patternDomain, "/etc/utmstack/compose.yml", "/etc/utmstack/application-prod.yml", "/etc/utmstack/vhost-elastic", "/etc/utmstack/vhost-utmstack", "/etc/utmstack/vhost-cerebro").Run()
	if err != nil {
		fmt.Println("[ERROR]: Setting domain =>", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting Domain FQDN ... Ok.")
	}

	// sysctl -w vm.swappiness=0
	err = exec.Command("sysctl", "-w", "vm.swappiness=0").Run()
	if err != nil {
		fmt.Println("[ERROR]: Setting vm.swappiness =>", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting vm.swappiness=0 ... Ok.")
	}

	// sysctl -w vm.overcommit_memory=1
	err = exec.Command("sysctl", "-w", "vm.overcommit_memory=1").Run()
	if err != nil {
		fmt.Println("[ERROR]: Setting vm.overcommit_memory", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting vm.overcommit_memory=1 ... Ok.")
	}

	// sysctl -w vm.max_map_count=4194304
	err = exec.Command("sysctl", "-w", "vm.max_map_count=4194304").Run()
	if err != nil {
		fmt.Println("[ERROR]: vm.max_map_count", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting vm.max_map_count=4194304 ... Ok.")
	}

	// echo 'vm.swappiness=0' >> /etc/sysctl.conf
	err = exec.Command("sh", "-c", `echo 'vm.swappiness=0' >> /etc/sysctl.conf`).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// echo 'vm.overcommit_memory=1' >> /etc/sysctl.conf
	err = exec.Command("sh", "-c", `echo 'vm.overcommit_memory=1' >> /etc/sysctl.conf`).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// echo 'vm.max_map_count=4194304' >> /etc/sysctl.conf
	err = exec.Command("sh", "-c", `echo 'vm.max_map_count=4194304' >> /etc/sysctl.conf`).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// TEST if APT is Busy
	// lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'
	outPID, _ = exec.Command("sh", "-c", `lsof /var/lib/dpkg/lock | awk 'NR==2{printf "%.i", $2}'`).Output()
	strPID = string(outPID)
	trimPID = strings.TrimRight(strPID, "\n")

	if trimPID != "" {
		fmt.Println("[ERROR] => [UTMSTACK]: APT-GET Services are busy, PID:", trimPID)
		os.Exit(1)
	} else {
		fmt.Println("# [UTMSTACK]: APT-GET Services ... Ok.")
	}

	// apt -y install docker-ce docker-ce-cli containerd.io
	fmt.Println("# [UTMSTACK]: Installing Docker Community Edition packages ...")
	cmd = exec.Command("sh", "-c", `apt -y install docker-ce docker-ce-cli containerd.io`)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// systemctl enable docker.service

	err = exec.Command("systemctl", "enable", "docker.service").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Enabling Docker as a permanent service ... Ok.")
	}

	// Pulling services images from Docker Hub
	fmt.Println("# [UTMSTACK]: Pulling services images from UTMStack Registry ... ~4.168 Gigabytes in total")
	// docker pull amazon/opendistro-for-elasticsearch:1.9.0
	fmt.Println("# [UTMSTACK]: Pulling Elasticsearch ... (~1.12GB, it takes some minutes depending on your bandwidth)")
	cmd = exec.Command("docker", "pull", "registry.utmstack.com:443/utm-elastic-aws")
	cmd.Stdout = os.Stdout
	cmd.Run()
	// docker pull registry.utmstack.com:443/utm-scanner11
	fmt.Println("# [UTMSTACK]: Pulling Scanner ... (~2.46GB, it takes some minutes depending on your bandwidth)")
	cmd = exec.Command("docker", "pull", "registry.utmstack.com:443/utm-scanner11")
	cmd.Stdout = os.Stdout
	cmd.Run()
	// docker pull utmstack/bionic-omp4war
	fmt.Println("# [UTMSTACK]: Pulling Ubuntu Bionic ... (~219MB, it takes some minutes depending on your bandwidth)")
	cmd = exec.Command("docker", "pull", "registry.utmstack.com:443/utm-bionic")
	cmd.Stdout = os.Stdout
	cmd.Run()
	// docker pull postgres:12-alpine
	fmt.Println("# [UTMSTACK]: Pulling PostgreSQL ... (~159MB, it takes some minutes depending on your bandwidth)")
	cmd = exec.Command("docker", "pull", "registry.utmstack.com:443/utm-postgres")
	cmd.Stdout = os.Stdout
	cmd.Run()
	// docker pull nginx:stable-alpine
	fmt.Println("# [UTMSTACK]: Pulling Nginx ... (~21.8MB, it takes some minutes depending on your bandwidth)")
	cmd = exec.Command("docker", "pull", "registry.utmstack.com:443/utm-nginx")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// TEST if tomcat.tar.gz EXISTS then remove
	if fileExists("/opt/tomcat.tar.gz") {
		strCMD = "rm /opt/tomcat.tar.gz /opt/tomcat.tar.gz.*"
		exec.Command("/bin/bash", "-c", strCMD).Run()
		fmt.Println("# [UTMSTACK]: Removed old version of file: tomcat.tar.gz ... Ok.")
	}

	// TEST if utm-stack.zip EXISTS then remove
	if fileExists("/opt/utm-stack.zip") {
		strCMD = "rm /opt/utm-stack.zip /opt/utm-stack.zip.*"
		exec.Command("/bin/bash", "-c", strCMD).Run()
		fmt.Println("# [UTMSTACK]: Removed old version of file: /opt/utm-stack.zip ... Ok.")
	}

	// Download tomcat.tar.gz
	// wget -O /opt/tomcat.tar.gz --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/tomcat.tar.gz
	fmt.Println("# [UTMSTACK]: Downloading tomcat.tar.gz... (~165MB, it takes some minutes depending on your bandwidth)")
	strCMD = "wget -O /opt/tomcat.tar.gz --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/tomcat.tar.gz"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Download utm-stack.zip
	// wget -O /opt/utm-stack.zip --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/utm-stack.zip
	fmt.Println("# [UTMSTACK]: Downloading utm-stack.zip... (~7.8MB, it takes some minutes depending on your bandwidth)")
	strCMD = "wget -O /opt/utm-stack.zip --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/utm-stack.zip"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Download updateUTM
	// wget -O /opt/updateUTM --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/updateUTM
	fmt.Println("# [UTMSTACK]: Downloading utm-stack.zip... (~2.2MB, it takes some minutes depending on your bandwidth)")
	strCMD = "wget -O /opt/updateUTM --ftp-user=upgrade --ftp-password='y#K=[swzk.(DQ7+[w;=#na/958yU$' ftp://registry.utmstack.com/updateUTM"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// docker swarm init --advertise-addr 127.0.0.1
	fmt.Println("# [UTMSTACK]: Initializing Docker Swarm Mode ...")
	cmd = exec.Command("docker", "swarm", "init", "--advertise-addr", "127.0.0.1")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Pulling services images from Docker Hub
	fmt.Println("# [UTMSTACK]: Creating Stack Infrastructure ...")

	//   docker volume create vol_scanner_data
	fmt.Println("# [UTMSTACK]: Creating Volume: vol_scanner_data ...")
	cmd = exec.Command("docker", "volume", "create", "vol_scanner_data")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// docker volume create vol_postgres_data
	fmt.Println("# [UTMSTACK]: Creating Volume: vol_postgres_data ...")
	cmd = exec.Command("docker", "volume", "create", "vol_postgres_data")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// docker network create --driver=overlay --attachable net_utmstack
	fmt.Println("# [UTMSTACK]: Creating Network: net_utmstack ...")
	cmd = exec.Command("docker", "network", "create", "--driver=overlay", "--attachable", "net_utmstack")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// tar -C /opt/ -zxf /opt/tomcat.tar.gz
	if fileExists("/opt/tomcat.tar.gz") {
		strCMD = "tar -C /opt/ -zxf /opt/tomcat.tar.gz"
		exec.Command("/bin/bash", "-c", strCMD).Run()
		fmt.Println("# [UTMSTACK]: Extracted tomcat.tar.gz ... Ok.")
	}

	// Tunning /opt/tomcat before deploy
	// ### Default root page configuration
	// sed '150i\ \ \ \ \ \ \ \ <Context docBase="${catalina.home}/webapps" path="" debug="0" reloadable="true"/>' /opt/tomcat/conf/server.xml > content.xml
	strCMD = "sed '150i\\ \\ \\ \\ \\ \\ \\ \\ <Context docBase=\"${catalina.home}/webapps\" path=\"\" debug=\"0\" reloadable=\"true\"/>' /opt/tomcat/conf/server.xml > content.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// cat content.xml > /opt/tomcat/conf/server.xml
	strCMD = "cat content.xml > /opt/tomcat/conf/server.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// rm content.xml
	err = exec.Command("rm", "content.xml").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Default root page configuration ... Ok.")
	}

	// ### Default error page configuration
	// sed '4709i\ \ <error-page>' /opt/tomcat/conf/web.xml > error1.xml
	strCMD = "sed '4709i\\ \\ <error-page>' /opt/tomcat/conf/web.xml > error1.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '4710i\ \ \ \ <error-code>404</error-code>' error1.xml > error2.xml
	strCMD = "sed '4710i\\ \\ \\ \\ <error-code>404</error-code>' error1.xml > error2.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '4711i\ \ \ \ <location>/index.html</location>' error2.xml > error3.xml
	strCMD = "sed '4711i\\ \\ \\ \\ <location>/index.html</location>' error2.xml > error3.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '4712i\ \ </error-page>' error3.xml > error4.xml
	strCMD = "sed '4712i\\ \\ </error-page>' error3.xml > error4.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// cat error4.xml > /opt/tomcat/conf/web.xml
	strCMD = "cat error4.xml > /opt/tomcat/conf/web.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// rm error1.xml error2.xml error3.xml error4.xml
	err = exec.Command("rm", "error1.xml", "error2.xml", "error3.xml", "error4.xml").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Default error page configuration ... Ok.")
	}

	// ### Adding Roles and Manager User
	// sed '44i\ \ <role rolename="admin-gui"/>' /opt/tomcat/conf/tomcat-users.xml > role1.xml
	strCMD = "sed '44i\\ \\ <role rolename=\"admin-gui\"/>' /opt/tomcat/conf/tomcat-users.xml > role1.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '45i\ \ <role rolename="admin-script"/>' role1.xml > role2.xml
	strCMD = "sed '45i\\ \\ <role rolename=\"admin-script\"/>' role1.xml > role2.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '46i\ \ <role rolename="manager-gui"/>' role2.xml > role3.xml
	strCMD = "sed '46i\\ \\ <role rolename=\"manager-gui\"/>' role2.xml > role3.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '47i\ \ <role rolename="manager-script"/>' role4.xml > role4.xml
	strCMD = "sed '47i\\ \\ <role rolename=\"manager-script\"/>' role3.xml > role4.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '48i\ \ <role rolename="manager-status"/>' role4.xml > role5.xml
	strCMD = "sed '48i\\ \\ <role rolename=\"manager-status\"/>' role4.xml > role5.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// sed '49i\ \ <user username="admin" password="blehblehbleh" roles="admin-gui,admin-script,manager-gui,manager-script,manager-status"/>' role5.xml > role6.xml
	strCMD = "sed '49i\\ \\ <user username=\"" + adminuser + "\" password=\"" + adminpass + "\" roles=\"admin-gui,admin-script,manager-gui,manager-script,manager-status\"/>' role5.xml > role6.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// cat role6.xml > /opt/tomcat/conf/tomcat-users.xml
	strCMD = "cat role6.xml > /opt/tomcat/conf/tomcat-users.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// rm role1.xml role2.xml ...
	err = exec.Command("rm", "role1.xml", "role2.xml", "role3.xml", "role4.xml", "role5.xml", "role6.xml").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Adding Roles and Manager User ... Ok.")
	}

	// Allowing Manager Web from everywhere
	// sed -i 's#127\\.\\d+\\.\\d+\\.\\d+|::1|0:0:0:0:0:0:0:1#.*#g' /opt/tomcat/webapps/manager/META-INF/context.xml
	strCMD = "sed -i 's#127\\\\.\\\\d+\\\\.\\\\d+\\\\.\\\\d+|::1|0:0:0:0:0:0:0:1#.*#g' /opt/tomcat/webapps/manager/META-INF/context.xml"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Allowing Manager Web from everywhere ... Ok.")
	}

	// cp /etc/utmstack/application-prod.yml /opt/tomcat/webapps/utmstack/WEB-INF/classes/config/application-prod.yml
	err = exec.Command("cp", "/etc/utmstack/application-prod.yml", "/opt/tomcat/webapps/utmstack/WEB-INF/classes/config/application-prod.yml").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// rm -Rf /opt/tomcat/webapps/docs /opt/tomcat/webapps/examples /opt/tomcat/webapps/host-manager /opt/tomcat/webapps/ROOT
	err = exec.Command("rm", "-Rf", "/opt/tomcat/webapps/docs", "/opt/tomcat/webapps/examples", "/opt/tomcat/webapps/host-manager", "/opt/tomcat/webapps/ROOT").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Cleaning Tomcat webapps ... Ok.")
	}

	// Checking and copying utm-stack.zip
	// unzip utm-stack.zip
	// cp -R utm-stack/* /opt/tomcat/webapps/
	// rm -Rf utm-stack/
	if _, err := os.Stat("/opt/utm-stack.zip"); os.IsNotExist(err) {
		fmt.Println("[ERROR] => [UTMSTACK]: Checking: /opt/utm-stack.zip failed, unable to deploy it")
	} else {
		err = exec.Command("unzip", "/opt/utm-stack.zip", "-d", "/opt").Run()
		strCMD = "cp -R /opt/utm-stack/* /opt/tomcat/webapps/"
		err = exec.Command("/bin/bash", "-c", strCMD).Run()
		err = exec.Command("rm", "-Rf", "/opt/utm-stack/").Run()
		fmt.Println("# [UTMSTACK]: Deploying utm-stack.zip ... Ok.")
	}

	// Creating & Deploying Stack Services
	fmt.Println("# [UTMSTACK]: Creating & Deploying Stack Services ...")
	// docker stack deploy --compose-file=/etc/utmstack/compose.yml utm
	cmd = exec.Command("docker", "stack", "deploy", "--compose-file=/etc/utmstack/compose.yml", "utm")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Wait 120 seconds to let services to deploy
	fmt.Println("# [UTMSTACK]: Waiting 2 minutes (120 seconds) to let services to deploy ...")
	var bar myBar
	bar.newBar(0, 120)
	for i := 0; i <= 120; i++ {
		time.Sleep(1 * time.Second)
		bar.playBar(int64(i))
	}
	bar.finishBar()

	fmt.Println("Starting Post-Install Configuration ...")

	// ########################################## POSTCONFIG ##########################################

	verified := ""
	if licensed {
		verified = "t"
	} else {
		verified = "f"
	}

	// INSERT utm_client TABLE
	// INSERT INTO "public"."utm_client" (client_name, client_domain, client_prefix, client_mail, client_user, client_pass, client_licence_creation, client_licence_expire, client_licence_id, client_licence_verified) VALUES (install_name, domain, install_prefix, install_mail, install_user, install_pass, install_licence_id, install_licence_verified);
	tmpAct, tmpExp := "", ""
	//strCMD = "echo \"INSERT INTO \"public\".\"utm_client\" (client_name,client_domain,client_prefix,client_mail,client_user,client_pass,client_licence_creation,client_licence_expire,client_licence_id,client_licence_verified) VALUES ('" + licName + "','" + domain + "','" + prefix + "','" + licMail + "','" + adminuser + "','" + adminpass + "','" + licAct + "','" + licExp + "','" + licKey + "','" + verified + "');\" >> /etc/utmstack/utm_client.sql"
	strCMD1 := "echo \"INSERT INTO \"public\".\"utm_client\" (client_name,client_domain,client_prefix,client_mail,client_user,client_pass,client_licence_creation,client_licence_expire,client_licence_id,client_licence_verified) VALUES ('" + licName + "','" + domain + "','" + prefix + "','" + licMail + "','" + adminuser + "','" + adminpass + "',"
	strCMD2 := ",'" + licKey + "','" + verified + "');\" >> /etc/utmstack/utm_client.sql"

	if licAct == "" {
		tmpAct = "NULL"
	} else {
		tmpAct = "'" + licAct + "'"
	}
	if licExp == "" {
		tmpExp = "NULL"
	} else {
		tmpExp = "'" + licExp + "'"
	}
	strCMD = strCMD1 + tmpAct + "," + tmpExp + strCMD2
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// cp /etc/utmstack/utm_client.sql /var/lib/docker/volumes/vol_postgres_data/_data/utm_client.sql
	cmd = exec.Command("cp", "/etc/utmstack/utm_client.sql", "/var/lib/docker/volumes/vol_postgres_data/_data/utm_client.sql")
	cmd.Run()

	// chown 70:70 /var/lib/docker/volumes/vol_postgres_data/_data/utm_client.sql
	strCMD = "chown 70:70 /var/lib/docker/volumes/vol_postgres_data/_data/utm_client.sql"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	}

	// docker ps -lqf "name=^utm_postgres"
	outPSQL, _ := exec.Command("docker", "ps", "-lqf", "name=^utm_postgres").Output()
	strPSQL := string(outPSQL)
	trimPSQL := strings.TrimRight(strPSQL, "\n")

	// docker exec $postgresID psql -h localhost -d utmstack -U <user> -f /var/lib/postgresql/data/utm_client.sql
	strCMD = "docker exec " + trimPSQL + " psql -h localhost -d utmstack -U " + adminuser + " -f /var/lib/postgresql/data/utm_client.sql"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Creating utm_client table ... Ok.")
	}

	// PUT _template/main_index
	// {"index_patterns":["index-*"],"settings":{"index.mapping.total_fields.limit":10000,"opendistro.index_state_management.policy_id":"main_index_policy","opendistro.index_state_management.rollover_alias":"index-prefix","number_of_shards":3,"number_of_replicas":0}}
	cfgCMD := "curl -X PUT 'http://localhost/_index_template/main_index' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"index_patterns\":[\"index-*\"],\"template\":{\"settings\":{\"index.mapping.total_fields.limit\":10000,\"opendistro.index_state_management.policy_id\":\"main_index_policy\",\"opendistro.index_state_management.rollover_alias\":\"index-" + string(prefix) + "\",\"number_of_shards\":3,\"number_of_replicas\":0}}}'"
	fmt.Println("Main Index String:", cfgCMD)
	outCfg, errCfg := exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _index_template/main_index ... Ok.")
	}

	// PUT _template/generic_index
	// {"index_patterns":["generic-*"],"settings":{"index.mapping.total_fields.limit":10000,"number_of_shards":1,"number_of_replicas":0}}
	cfgCMD = "curl -X PUT 'http://localhost/_index_template/generic_index' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"index_patterns\":[\"generic-*\"],\"template\":{\"settings\":{\"index.mapping.total_fields.limit\":10000,\"number_of_shards\":1,\"number_of_replicas\":0}}}'"
	fmt.Println("Generic String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _index_template/generic_index ... Ok.")
	}

	// PUT _template/dc_index
	// {"index_patterns":["dc-*"],"settings":{"number_of_shards":1,"number_of_replicas":0}}
	cfgCMD = "curl -X PUT 'http://localhost/_index_template/dc_index' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"index_patterns\":[\"dc-*\"],\"template\":{\"settings\":{\"number_of_shards\":1,\"number_of_replicas\":0}}}'"
	fmt.Println("DC String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _index_template/dc_index ... Ok.")
	}

	// PUT _template/utmstack_index
	// {"index_patterns":["utmstack-*"],"settings":{"number_of_shards":1,"number_of_replicas":0}}
	cfgCMD = "curl -X PUT 'http://localhost/_index_template/utmstack_index' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"index_patterns\":[\"utmstack-*\"],\"template\":{\"settings\":{\"number_of_shards\":1,\"number_of_replicas\":0}}}'"
	fmt.Println("UtmStack String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _index_template/utmstack_index ... Ok.")
	}

	// PUT _snapshot/main_index
	// {"type":"fs","settings":{"location":"main_index"}}
	cfgCMD = "curl -X PUT 'http://localhost/_snapshot/main_index' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"type\":\"fs\",\"settings\":{\"location\":\"main_index\"}}'"
	fmt.Println("Repo String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _snapshot/main_index ... Ok.")
	}

	// PUT _opendistro/_ism/policies/main_index_policy
	// {"policy":{"description":"MainIndexLifecycle","default_state":"ingest","states":[{"name":"ingest","actions":[{"rollover":{"min_doc_count":30000000,"min_size":"15gb"}}],"transitions":[{"state_name":"search"}]},{"name":"search","actions":[{"snapshot":{"repository":"main_index","snapshot":"incremental"}}],"transitions":[{"state_name":"read","conditions":{"min_index_age":"30d"}}]},{"name":"read","actions":[{"force_merge":{"max_num_segments":1}},{"snapshot":{"repository":"main_index","snapshot":"incremental"}}],"transitions":[]}]}}
	cfgCMD = "curl -X PUT 'http://localhost/_opendistro/_ism/policies/main_index_policy' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"policy\":{\"description\":\"MainIndexLifecycle\",\"default_state\":\"ingest\",\"states\":[{\"name\":\"ingest\",\"actions\":[{\"rollover\":{\"min_doc_count\":30000000,\"min_size\":\"15gb\"}}],\"transitions\":[{\"state_name\":\"search\"}]},{\"name\":\"search\",\"actions\":[{\"snapshot\":{\"repository\":\"main_index\",\"snapshot\":\"incremental\"}}],\"transitions\":[{\"state_name\":\"read\",\"conditions\":{\"min_index_age\":\"30d\"}}]},{\"name\":\"read\",\"actions\":[{\"force_merge\":{\"max_num_segments\":1}},{\"snapshot\":{\"repository\":\"main_index\",\"snapshot\":\"incremental\"}}],\"transitions\":[]}]}}'"
	fmt.Println("ISM String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _opendistro/_ism/policies/main_index_policy ... Ok.")
	}

	// PUT index-pnb-000001 {}
	cfgCMD = "curl -X PUT 'http://localhost/index-" + string(prefix) + "-000001' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{}'"
	fmt.Println("Index String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT index-" + string(prefix) + "-000001 ... Ok.")
	}

	// POST _aliases
	// {"actions":[{"add":{"index":"index-prefix-000001","alias":"index-prefix"}}]}
	cfgCMD = "curl -X POST 'http://localhost/_aliases' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"actions\":[{\"add\":{\"index\":\"index-" + string(prefix) + "-000001\",\"alias\":\"index-" + prefix + "\"}}]}'"
	fmt.Println("Alias String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: POST _aliases ... Ok.")
	}

	// Adding UTM-GEOIP Snapshot.
	// PUT _snapshot/repo-utm-geoip
	// {"type":"fs","settings":{"location":"utm-geoip"}}
	cfgCMD = "curl -X PUT 'http://localhost/_snapshot/repo-utm-geoip' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"type\":\"fs\",\"settings\":{\"location\":\"utm-geoip\"}}'"
	fmt.Println("Repo String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _snapshot/utm-geoip ... Ok.")
	}

	// Restoring UTM-GEOIP Snapshot.
	// POST _snapshot/repo-utm-geoip/utm-geoip/_restore
	// {"include_global_state": false}
	cfgCMD = "curl -X POST 'http://localhost/_snapshot/repo-utm-geoip/utm-geoip/_restore' -H 'Host:elastic." + string(domain) + "' -H 'Content-Type: application/json' -d '{\"include_global_state\": false}'"
	fmt.Println("Repo String:", cfgCMD)
	outCfg, errCfg = exec.Command("/bin/bash", "-c", cfgCMD).Output()
	fmt.Println("Output:", string(outCfg))
	if errCfg != nil {
		fmt.Println("[ERROR]:", errCfg)
	} else {
		fmt.Println("# [UTMSTACK]: PUT _snapshot/utm-geoip ... Ok.")
	}

	// DOUBLE CHECK for WAR DEPLOYMENT
	// curl -ku admin:AdminPass https://www.utmclient.utmstack.com/manager/text/list
	// curl -ku admin:AdminPass https://www.utmclient.utmstack.com/manager/text/start?path=/utmstack
	fmt.Println("# [UTMSTACK]: Verify and/or redeploy Application ...")
	startCMD := "curl -ku " + adminuser + ":" + adminpass + " https://www.utmclient.utmstack.com/manager/text/start?path=/utmstack"
	cmd = exec.Command("/bin/bash", "-c", startCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()

	//(M)aster and Infrastructure (UTMStack)
	fmt.Println("# [UTMSTACK]: Infrastructure and Services are installed, time for Master Data Collector ...")
	answer := ""
	opt := ""
	fmt.Println("# [UTMSTACK]: Are you sure to proceed?")
	for {
		fmt.Println("(Y)es to continue with Master Data Collector.")
		fmt.Println("(N)o to abort doing nothing.")
		fmt.Print("Your answer is?: ")
		fmt.Scanln(&answer)
		opt = string(answer)
		if opt == "y" || opt == "Y" {
			break
		}
		if opt == "n" || opt == "N" {
			fmt.Println("# [UTMSTACK]: Exiting the installer ...")
			os.Exit(1)
		}
		answer = ""
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}

	fmt.Println("# [UTMSTACK]: Installing Master Data Collector ...")
	// stack install [master|probe|aio]
	stackCMD := "/usr/local/bin/stack install aio"

	// cp -R .aws /root/
	strCMD = "cp -R .aws /root/"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Copying .aws configuration ... Ok.")
	}

	// wget -O /usr/local/bin/stack https://updates.utmstack.com/assets/stack
	fmt.Println("# [UTMSTACK]: Downloading installer ...")
	cmd = exec.Command("wget", "-O", "/usr/local/bin/stack", "https://updates.utmstack.com/assets/stack")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// chmod +x /usr/local/bin/stack
	strCMD = "chmod +x /usr/local/bin/stack"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting executable permission to installer ... Ok.")
	}

	cmd = exec.Command("/bin/bash", "-c", stackCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()
	return
}

func utmUninstall() { // Uninstaller function.
	answer := ""
	opt := ""
	fmt.Println("# [UTMSTACK]: This is the UTM Stack UNINSTALLER ...")
	fmt.Println("# [UTMSTACK]: Are you sure to proceed?")
	for {
		fmt.Println("(Y)es to continue with the uninstallation.")
		fmt.Println("(N)o to abort doing nothing.")
		fmt.Print("Your answer is?: ")
		fmt.Scanln(&answer)
		opt = string(answer)
		if opt == "y" || opt == "Y" {
			fmt.Println("# [UTMSTACK]: The uninstaller proceeds ...")
			break
		}
		if opt == "n" || opt == "N" {
			fmt.Println("# [UTMSTACK]: Exiting the unistaller ...")
			os.Exit(1)
		}
		answer = ""
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}

	// docker swarm leave --force
	fmt.Println("# [UTMSTACK]: Leaving Docker Swarm Stack ...")
	cmd := exec.Command("docker", "swarm", "leave", "--force")
	cmd.Stdout = os.Stdout
	cmd.Run()

	//docker volume rm vol_scanner_data vol_postgres_data
	fmt.Println("# [UTMSTACK]: Removing Docker Volumes ...")
	cmd = exec.Command("docker", "volume", "rm", "vol_scanner_data", "vol_postgres_data")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// rm -Rf /var/lib/docker/volumes/vol_* /var/lib/docker/volumes/utm_*
	strCMD := "rm -Rf /var/lib/docker/volumes/vol_* /var/lib/docker/volumes/utm_*"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Run()

	// // add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs) main restricted universe multiverse"
	// fmt.Println("# [UTMSTACK]: Removing the Ubuntu repository, branch UBUNTU ...")
	// cmd = exec.Command("sh", "-c", `add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs) main restricted universe multiverse"`)
	// cmd.Run()

	// // add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-updates main restricted universe multiverse"
	// fmt.Println("# [UTMSTACK]: Removing the Ubuntu repository, branch UBUNTU-UPDATES ...")
	// cmd = exec.Command("sh", "-c", `add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-updates main restricted universe multiverse"`)
	// cmd.Run()

	// // add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-security main restricted universe multiverse"
	// fmt.Println("# [UTMSTACK]: Removing the Ubuntu repository, branch UBUNTU-SECURITY ...")
	// cmd = exec.Command("sh", "-c", `add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-security main restricted universe multiverse"`)
	// cmd.Run()

	// // add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-backports main restricted universe multiverse"
	// fmt.Println("# [UTMSTACK]: Removing the Ubuntu repository, branch UBUNTU-BACKPORTS ...")
	// cmd = exec.Command("sh", "-c", `add-apt-repository -r "deb [arch=amd64] http://us.archive.ubuntu.com/ubuntu $(lsb_release -cs)-backports main restricted universe multiverse"`)
	// cmd.Run()

	// // add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
	// fmt.Println("# [UTMSTACK]: Removing the stable repository for Docker ...")
	// cmd = exec.Command("sh", "-c", `add-apt-repository -r "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"`)
	// cmd.Run()

	// apt -y remove docker-ce docker-ce-cli containerd.io && apt -y purge docker-ce docker-ce-cli containerd.io && apt autoremove
	fmt.Println("# [UTMSTACK]: Removing Docker packages ...")
	strCMD = "apt -y remove docker-ce docker-ce-cli containerd.io && apt -y purge docker-ce docker-ce-cli containerd.io"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Run()

	// apt -y remove apt-transport-https ca-certificates curl gnupg-agent software-properties-common && apt -y purge apt-transport-https ca-certificates curl gnupg-agent software-properties-common && apt autoremove
	fmt.Println("# [UTMSTACK]: Removing Support packages ...")
	strCMD = "apt -y remove apt-transport-https ca-certificates curl gnupg-agent software-properties-common && apt -y purge apt-transport-https ca-certificates curl gnupg-agent software-properties-common && apt autoremove"
	cmd = exec.Command("/bin/bash", "-c", strCMD)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Run()

	err := exec.Command("rm", "/opt/tomcat.tar.gz").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Removing /opt/tomcat.tar.gz ... Ok.")
	}

	err = exec.Command("rm", "/opt/utm-stack.zip").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Removing /opt/utm-stack.zip ... Ok.")
	}

	err = exec.Command("rm", "-Rf", "/opt/tomcat").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Removing /opt/tomcat ... Ok.")
	}

	fmt.Println("# [UTMSTACK]: There will be a remnant inside some folders and files:")
	fmt.Println("# - /utmstack/data (Data from Nodes)")
	fmt.Println("# - /utmstack/repo (Backups of your Data)")
	fmt.Println("# - /opt/tomcat (Application Folder)")
	fmt.Println("# - /etc/apt/sources.list (Docker repository)")
	fmt.Println("# - /var/lib/docker/* (Docker Folders)")
	fmt.Println("# [UTMSTACK]: If you are absolutely sure you won't need this content for the future, just delete these folders and remove lines concerning Docker Repository.")
	return
}

func proxyProbe() { // Data Collector
	scanner := bufio.NewScanner(os.Stdin)
	// Ask for PostgreSQL credentials
	fmt.Println("# [UTMSTACK]: Enter credentials for PostgreSQL user.")
	fmt.Print("Your PostgreSQL Username: ")
	scanner.Scan()
	adminuser := scanner.Text()

	fmt.Print("Your PostgreSQL Password: ")
	scanner.Scan()
	adminpass := scanner.Text()

	// stack install [master|probe|aio]
	stackCMD := "/usr/local/bin/stack install probe"

	// mkdir /etc/utmstack/
	if err := os.Mkdir("/etc/utmstack", 0755); err != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was a problem creating: /etc/utmstack/", "| Error:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Folder: /etc/utmstack Created ... Ok.")
	}

	// sed -i "s|utmstack_user|$adminuser|g" compose.yml application-prod.yml
	patternUser := "s|utmstack_user|" + adminuser + "|g"
	err := exec.Command("sed", "-i", patternUser, "/etc/utmstack/config.json").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting Admin username ... Ok.")
	}

	// sed -i "s|utmstack_pass|$adminpass|g" compose.yml application-prod.yml
	patternPass := "s|utmstack_pass|" + adminpass + "|g"
	err = exec.Command("sed", "-i", patternPass, "/etc/utmstack/config.json").Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting Admin password ... Ok.")
	}

	// cp -R .aws /root/
	strCMD := "cp -R .aws /root/"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Copying .aws configuration ... Ok.")
	}

	// wget -O /usr/local/bin/stack https://updates.utmstack.com/assets/stack
	fmt.Println("# [UTMSTACK]: Downloading installer ...")
	cmd := exec.Command("wget", "-O", "/usr/local/bin/stack", "https://updates.utmstack.com/assets/stack")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// chmod +x /usr/local/bin/stack
	strCMD = "chmod +x /usr/local/bin/stack"
	err = exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Setting executable permission to installer ... Ok.")
	}

	cmd = exec.Command("/bin/bash", "-c", stackCMD)
	cmd.Stdout = os.Stdout
	cmd.Run()
	return
}

func agreeEULA() {
	answer := ""
	opt := ""
	for {
		fmt.Println("# ----------------------------------------------")
		fmt.Println("# [UTMSTACK]: End User License Agreement (EULA).")
		fmt.Println("# ----------------------------------------------")
		fmt.Println("# [UTMSTACK]: How do you want to proceed?")
		fmt.Println("(A)ccept the agreement and continue.")
		fmt.Println("(R)eview the UTMStack License. (Hit 'Q' or 'q' to leave the editor)")
		fmt.Println("(C)ancel the Setup and do Nothing.")
		fmt.Print("Your answer is?: ")
		fmt.Scanln(&answer)
		opt = string(answer)
		if opt == "a" || opt == "A" {
			fmt.Println("# [UTMSTACK]: Since you accept the EULA, the setup proceeds ...")
			break
		}
		if opt == "r" || opt == "R" {
			cmd := exec.Command("less", "/etc/utmstack/License-Private-UTMStack.md")
			cmd.Stdout = os.Stdout
			cmd.Run()
			continue
		}
		if opt == "c" || opt == "C" {
			fmt.Println("# [UTMSTACK]: Exiting the setup ...")
			os.Exit(0)
		}
		answer = ""
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}
}

func selectProduct() string { // Select Installation Option
	answer := ""
	for {
		fmt.Println("# ----------------------------------------")
		fmt.Println("# [UTMSTACK]: Select Product Setup Option.")
		fmt.Println("# ----------------------------------------")
		fmt.Println("(M)aster and Infrastructure (UTMStack).")
		fmt.Println("(P)roxy for Data Collection (Just Probe).")
		fmt.Println("(C)ancel the Setup and do Nothing.")
		fmt.Println("# ----------------------------------------")
		fmt.Print("Your answer is?: ")
		fmt.Scanln(&answer)
		opt := string(answer)
		if opt == "m" || opt == "M" {
			return "m"
		}
		if opt == "p" || opt == "P" {
			return "p"
		}
		if opt == "c" || opt == "C" {
			os.Exit(0)
		}
		answer = ""
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}
}

func inputData() { // Inputs for adminuser, adminpass, domain, prefix, licName, licMail, licKey
	scanner := bufio.NewScanner(os.Stdin)
	rawprefix := ""
	install := ""
	left := 0
	for {
		adminuser, adminpass, domain, rawprefix, licName, licMail, licKey, install = "", "", "", "", "", "", "", ""
		licensed = false

		// Ask for credentials, domain and License
		fmt.Println("# [UTMSTACK]: Enter credentials for administrator user.")
		fmt.Print("Your Admin Username: ")
		scanner.Scan()
		adminuser = scanner.Text()

		fmt.Print("Your Admin Password: ")
		scanner.Scan()
		adminpass = scanner.Text()

		fmt.Print("Your Domain FQDN is: ")
		scanner.Scan()
		domain = scanner.Text()

		fmt.Print("Your Name Prefix (ex. mycompany) (10 chars MAX) is: ")
		scanner.Scan()
		rawprefix = scanner.Text()

		if version == "l" {
			fmt.Println("# [UTMSTACK]: ----------- LICENSE DETAILS ----------")
			fmt.Print("Your LICENSE Key is: ")
			scanner.Scan()
			licKey = scanner.Text()
		}

		if rawprefix == "" {
			prefix = randomString(10)
		} else {
			prefix = trimPrefix(rawprefix)
		}

		if adminuser == "" || adminpass == "" || domain == "" {
			fmt.Print("[ERROR] => [UTMSTACK]: Empty Fields: | ")
			if adminuser == "" {
				fmt.Print("adminuser | ")
			}
			if adminpass == "" {
				fmt.Print("adminpass | ")
			}
			if domain == "" {
				fmt.Print("domain | ")
			}
			fmt.Println(" ")
			fmt.Println("# [UTMSTACK]: These inputs MUST have a non-empty value. Starting over ...")
			fmt.Println("# [UTMSTACK]: -----------------------------------------------------------")
			continue
		}

		if version == "l" {
			// Checking days left on License
			left = licenseDays(licKey)
		}

		if left > 0 {
			install = "LICENSED VERSION"
			licensed = true

		} else {
			install = "FREE VERSION"
			licensed = false
		}

		for {
			fmt.Println("# [UTMSTACK]: ---------------------------------------")
			fmt.Println("# [UTMSTACK]: VERIFY YOUR INPUTS ...")
			fmt.Println("# [UTMSTACK]: ---------------------------------------")
			fmt.Println("# [UTMSTACK]: Admin Username:", adminuser)
			fmt.Println("# [UTMSTACK]: Admin Password:", adminpass)
			fmt.Println("# [UTMSTACK]: Domain FQDN is:", domain)
			fmt.Println("# [UTMSTACK]: Company Prefix:", prefix)
			if licensed == true {
				fmt.Println("# [UTMSTACK]: ---------------------------------------")
				fmt.Println("# [UTMSTACK]: LICENSE Key is:", licKey)
				fmt.Println("# [UTMSTACK]: ---------------------------------------")
				fmt.Println("# [UTMSTACK]: LICENSE Validity:", left, "days left.")
				fmt.Println("# [UTMSTACK]: Install type is:", install)
			}
			fmt.Println("# [UTMSTACK]: ---------------------------------------")
			fmt.Println("# [UTMSTACK]: Is this correct to proceed?")
			fmt.Println("(Y)es to continue")
			fmt.Println("(N)o to repeat the INPUTS")
			fmt.Println("(C)ancel the Setup and do Nothing.")
			fmt.Print("Your answer is?: ")
			answer := ""
			fmt.Scanln(&answer)
			opt := string(answer)
			if opt == "y" || opt == "Y" {
				return
			}
			if opt == "n" || opt == "N" {
				fmt.Println("# [UTMSTACK]: Ok. Starting over ...")
				break
			}
			if opt == "c" || opt == "C" {
				os.Exit(0)
			}
			fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
			continue
		}
	}
}

func licenseGet(key string) licenseJSON {
	var result licenseJSON
	// curl -k "https://utmstack.com/wp-json/lmfwc/v2/licenses/[key]?consumer_key=ck_ad89e93e4d5ea9435c5736cd5c7a2ebcf0e26f3c&consumer_secret=cs_c5fd29d6a453bb0d3f318620e53a00ea1fb555aa"
	verify := `curl -k "https://utmstack.com/wp-json/lmfwc/v2/licenses/` + key + `?consumer_key=ck_ad89e93e4d5ea9435c5736cd5c7a2ebcf0e26f3c&consumer_secret=cs_c5fd29d6a453bb0d3f318620e53a00ea1fb555aa"`
	outChk, errChk := exec.Command("/bin/bash", "-c", verify).Output()
	errJSON := json.Unmarshal(outChk, &result)
	if errChk != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was an error trying to verify License:", errChk)
	}
	if errJSON != nil {
		fmt.Println("[ERROR] => [UTMSTACK]: There was an error trying to decode License:", errJSON)
	}
	return result
}

func licenseDays(key string) int {
	live := 0
	now := time.Now()
	nowDate := now.Format("2006-01-02 15:04:05")
	license := licenseGet(key)
	if license.Success {
		if license.Data.Status == 2 {
			license.printString()
			live = dateSub(nowDate, license.Data.ExpiresAt)
		} else {
			fmt.Println("[ERROR] => [UTMSTACK]: License Status Invalid:", string(live))
		}
	} else {
		fmt.Println("[ERROR] => [UTMSTACK]: License Validity Check Failed.")
		fmt.Println("   Code:", string(license.Code))
		fmt.Println("Message:", string(license.Message))
	}
	return live
}

func dateSub(active string, expire string) int {
	dateActive, _ := time.Parse("2006-01-02 15:04:05", active)
	dateExpire, _ := time.Parse("2006-01-02 15:04:05", expire)
	days := dateExpire.Sub(dateActive).Hours() / 24
	return int(days)
}

func (l licenseJSON) printString() {
	licAct = l.Data.CreatedAt
	licExp = l.Data.ExpiresAt
	status := ""
	switch l.Data.Status {
	case 2:
		status = "Delivered"
	case 3:
		status = "Active"
	case 4:
		status = "Inactive"
	default:
		status = "UnKnown"
	}
	fmt.Println("    License Key:", string(l.Data.LicenseKey))
	fmt.Println("     Expires At:", string(l.Data.ExpiresAt))
	fmt.Println("      Valid For:", strconv.Itoa(l.Data.ValidFor))
	fmt.Println("         Status:", status)
	fmt.Println("Times Activated:", strconv.Itoa(l.Data.TimesActivated))
	fmt.Println("     Created At:", string(l.Data.CreatedAt))
	fmt.Println("     Updated At:", string(l.Data.UpdatedAt))
}

func (bar *myBar) newBar(start, total int64) { // Progress NewBar
	bar.cur = start
	bar.total = total
	if bar.graph == "" {
		bar.graph = "#"
	}
	bar.percent = bar.getPercent()
	for i := 0; i < int(bar.percent); i += 2 {
		bar.rate += bar.graph // initial progress position
	}
}

func (bar *myBar) getPercent() int64 { // Progress getPercent
	return int64(float32(bar.cur) / float32(bar.total) * 100)
}

func (bar *myBar) playBar(cur int64) { // Progress Play
	bar.cur = cur
	last := bar.percent
	bar.percent = bar.getPercent()
	if bar.percent != last && bar.percent%2 == 0 {
		bar.rate += bar.graph
	}
	fmt.Printf("\r[%-50s]%3d%% %8d/%d", bar.rate, bar.percent, bar.cur, bar.total)
}

func (bar *myBar) finishBar() { // Progress Finish
	bar.percent = 0
	bar.cur = 0
	bar.total = 0
	bar.rate = ""
	bar.graph = ""
	fmt.Println()
}

func trimPrefix(prefix string) string {
	trimPrefix := strings.Replace(prefix, " ", "", -1)
	lowerPrefix := strings.ToLower(trimPrefix)
	if len(lowerPrefix) > 10 {
		return lowerPrefix[:10]
	}
	return lowerPrefix
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func copyFiles() { // Create installation files
	// File utmstack_utm_client.sql
	utmClient := `CREATE TABLE IF NOT EXISTS "public"."utm_client" ("id" SERIAL PRIMARY KEY, "client_name" varchar(100), "client_domain" varchar(100), "client_prefix" varchar(10), "client_mail" varchar(100), "client_user" varchar(50), "client_pass" varchar(50), "client_licence_creation" timestamp(0), "client_licence_expire" timestamp(0), "client_licence_id" varchar(100), "client_licence_verified" bool NOT NULL);
`
	// File utmstack_compose.yml
	utmCompose := `version: "3.7"
### STACK VOLUMES
volumes:
  # POSTGRESQL
  vol_postgres_data:
    external: true
  # Scanner
  vol_scanner_data:
    external: true
### STACK NETWORKS
networks:
  net_utmstack:
    external: true
### STACK SERVICES
services:
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### NGINX, LOAD BALANCER & REVERSE PROXY
  nginx:
    image: registry.utmstack.com:443/utm-nginx
    volumes:
      - /etc/utmstack/vhost-elastic:/etc/nginx/conf.d/default.conf
      - /etc/utmstack/vhost-utmstack:/etc/nginx/conf.d/utmstack.conf
      - /etc/utmstack/vhost-cerebro:/etc/nginx/conf.d/cerebro.conf
      - /etc/utmstack/single.crt:/etc/nginx/single.crt
      - /etc/utmstack/single.key:/etc/nginx/single.key
    networks:
      - net_utmstack
    ports:
      - "80:80"     # HTTP
      - "443:443"   # HTTPS
    deploy:
      placement:
        constraints: [node.role==manager]
      mode: global
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### ELASTIC SEARCH, Single Nodes
  es-elastic:
    image: registry.utmstack.com:443/utm-elastic-aws
    volumes:
      - /utmstack/data/single:/usr/share/elasticsearch/data
      - /utmstack/repo:/usr/share/elasticsearch/backup
      - /etc/localtime:/etc/localtime:ro
    networks:
      - net_utmstack
    environment:
      - node.name=es-elastic
      - cluster.name=es-cluster
      - network.host=_eth0_
      - opendistro_security.disabled=true
      - discovery.type=single-node
      - path.repo=/usr/share/elasticsearch/backup
      - "ES_JAVA_OPTS=-XmsJVM_CORDm -XmxJVM_CORDm"
    deploy:
      endpoint_mode: dnsrr
      mode: global
      placement:
        constraints: [node.role==manager]
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### CEREBRO, ELASTICSEARCH WEB ADMIN TOOL
  cerebro:
    image: registry.utmstack.com:443/utm-cerebro
    volumes:
      - /etc/localtime:/etc/localtime:ro
    networks:
      - net_utmstack
    command:
      - -Dhosts.0.host=http://utm_es-elastic:9200
    deploy:
      placement:
        constraints: [node.role==manager]
      mode: global
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### POSTGRES, DATABASE SERVER
  postgres:
    image: registry.utmstack.com:443/utm-postgres
    ports:
      - 5432:5432
    volumes:
      - vol_postgres_data:/var/lib/postgresql/data
      - /etc/localtime:/etc/localtime:ro
    environment:
      POSTGRES_USER: utmstack_user
      POSTGRES_PASSWORD: utmstack_pass
      POSTGRES_DB: utmstack
    command: ["-c", "max_connections=1000"]
    networks:
      - net_utmstack
    deploy:
      mode: global
      placement:
        constraints: [node.role==manager]
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### SCANNER & MANAGER
  scanner:
    image: registry.utmstack.com:443/utm-scanner11
    volumes:
      - vol_scanner_data:/data
      - /etc/localtime:/etc/localtime:ro
    networks:
      - net_utmstack
    ports:
      - "8888:5432"
    environment:
      USERNAME: utmstack_user
      PASSWORD: utmstack_pass
      DB_PASSWORD: utmstack_pass
      HTTPS: 0
    deploy:
      resources:
        limits:
          cpus: '2.0'
      mode: global
      placement:
        constraints: [node.role==manager]
# -------------------------------------------------- NEW SERVICE -------------------------------------------------- #
  ### TOMCAT, JAVA WEBSERVER
  tomcat:
    image: registry.utmstack.com:443/utm-bionic
    volumes:
      - /opt/tomcat:/opt/tomcat
      - /etc/localtime:/etc/localtime:ro
    networks:
      - net_utmstack
    environment:
      JRE_HOME: "/opt/tomcat/bin/jre"
      JAVA_HOME: "/opt/tomcat/bin/jre"
      CATALINA_BASE: "/opt/tomcat"
      CATALINA_HOME: "/opt/tomcat"
      LD_LIBRARY_PATH: "/usr/lib/x86_64-linux-gnu"
      CATALINA_OPTS: "-XmsJAVA_MINm -XmxJAVA_MAXm -XX:+CMSClassUnloadingEnabled -XX:+UseConcMarkSweepGC -XX:+CMSPermGenSweepingEnabled"
    command: ["/opt/tomcat/bin/catalina.sh", "run"]
    deploy:
      placement:
        constraints: [node.role==manager]
      mode: global`

	// File utmstack_application-prod.yml
	appProd := `logging:
  level:
    ROOT: INFO
    com.park.utmstack: INFO
    io.github.jhipster: INFO
spring:
  devtools:
    restart:
      enabled: false
    livereload:
      enabled: false
  datasource: # For Postgresql database
    type: com.zaxxer.hikari.HikariDataSource
    url: jdbc:postgresql://utm_postgres:5432/utmstack
    username: utmstack_user
    password: utmstack_pass
    hikari:
      poolName: Hikari
      auto-commit: false
  jpa:
    database-platform: io.github.jhipster.domain.util.FixedPostgreSQL82Dialect # For Postgresql database
    database: POSTGRESQL # For Postgresql database
    show-sql: false
    properties:
      hibernate.id.new_generator_mappings: true
      hibernate.connection.provider_disables_autocommit: true
      hibernate.cache.use_second_level_cache: false
      hibernate.cache.use_query_cache: false
      hibernate.generate_statistics: true
      hibernate.jdbc.batch_size: 15
      hibernate.order_inserts: true
  elasticsearch:
    rest:
      uris: http://utm_es-elastic:9200
    jest:
      uris: http://utm_es-elastic:9200
  liquibase:
    contexts: prod
  mail:
    host: ${utmstack.mail.host} #mail.atlasinside.com
    port: ${utmstack.mail.port} #587
    username: ${utmstack.mail.username} #demo@utmvault.com
    password: ${utmstack.mail.password} #e235g7b558bxbwd7
    protocol: ${utmstack.mail.protocol} #smtp
    properties.mail.smtp:
      auth: ${utmstack.mail.properties.mail.smtp.auth} #true
      starttls.enable: ${utmstack.mail.properties.mail.smtp.starttls.enable} #true
      ssl.trust: ${utmstack.mail.properties.mail.smtp.ssl.trust} #mail.atlasinside.com
  thymeleaf:
    cache: true
server:
  port: 8080
  compression:
    enabled: true
    mime-types: text/html,text/xml,text/plain,text/css, application/javascript, application/json
    min-response-size: 1024
jhipster:
  http:
    version: V_1_1 # To use HTTP/2 you will need SSL support (see above the "server.ssl" configuration)
    cache: # Used by the CachingHttpHeadersFilter
      timeToLiveInDays: 1461
  cors:
    allowed-origins: "*"
    allowed-methods: "*"
    allowed-headers: "*"
    exposed-headers: "X-utmvaultApp-code,X-utmvaultApp-error,Origin,X-Requested-With,Content-Type,Accept,x-auth-token,Authorization,Link,X-Total-Count"
    allow-credentials: true
    max-age: 1800
  security:
    authentication:
      jwt:
        base64-secret: MzY5Mzg0ZTViYzUwMzc0ODhlYmE2ZDUxN2Y5OGVmNDM3ZDE4ZmY1YWUwMzliNmM4ZWU2NDU1ZTc2YTg0MWRmZjcyZDQxOTNhMTJkODgwZjg2YWVhNWNlZjk5ZmY5MzYwOTQxOTI4MDRhMDE5ZGUzYjliYTYzN2Y1OTU0YTQzNjU=
        token-validity-in-seconds: 86400
        token-validity-in-seconds-for-remember-me: 2592000
  mail: # specific JHipster mail property, for standard properties see MailProperties
    from: ${utmstack.mail.from}
    base-url: ${utmstack.mail.baseUrl}
  metrics:
    logs: # Reports metrics in the logs
      enabled: true
      report-frequency: 60 # in seconds
  logging:
    logstash: # Forward logs to logstash over a socket, used by LoggingConfiguration
      enabled: false
      host: localhost
      port: 5000
      queue-size: 512
application:
  alert-notification-mails: ${utmstack.mail.addressToNotify}
  openvas: # Scan Manager credentials
    host: utm_scanner
    port: 9390
    user: utmstack_user
    password: utmstack_pass
    datasource:
      url: jdbc:postgresql://utm_scanner:8888/gvmd
      username: gvm
      password: utmstack_pass
  chart-builder: # Chart builder configuration
    elasticsearch-url: http://utm_es-elastic:9200
    ip-info-index-name: utm-geoip
  active-directory:
    winlogbeat-index-pattern: index-*
    active-directory-index-pattern: dc-*
  application-event:
    index: utmstack-logs
  tools:
    - name: "Vulnerability Management System"
      signature-amount: ""
      last-update: ""
    - name: "HIPS"
      signature-amount: ""
      last-update: ""
    - name: "NIDS"
      signature-amount: ""
      last-update: ""
    - name: "Correlation Engine"
      signature-amount: ""
      last-update: ""
  incident-response:
    asset-verification-interval: 300
  notification:
    twilio:
      accountSID: ${utmstack.twilio.accountSid}
      authToken: ${utmstack.twilio.authToken}
      phoneNumber: ${utmstack.twilio.number}`

	// File config.json
	probeJSON := `{
  "postgresql": {
    "host": "localhost",
    "user": "utmstack_user",
    "password": "utmstack_pass",
    "database": "utmstack",
    "port": 5432
  }
}`

	// File vhost-elastic
	vhostElastic := `server {
  listen 80;
  server_name elastic.utmclient.utmstack.com;
  resolver 127.0.0.11 valid=600s ipv6=off;
  set $elastic http://utm_es-elastic:9200;
  client_max_body_size 0;
  chunked_transfer_encoding on;
  location / {
    proxy_pass $elastic;
    proxy_set_header Host $http_host;   # required for docker client's sake
    proxy_set_header X-Real-IP $remote_addr; # pass on real client's IP
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 900;
  }
}`

	// File vhost-cerebro
	vhostCerebro := `server {
    listen 80;
    server_name cerebro.utmclient.utmstack.com;
    resolver 127.0.0.11 valid=600s ipv6=off;
    set $cerebro http://utm_cerebro:9000;
    client_max_body_size 0;
    chunked_transfer_encoding on;
    location / {
      proxy_pass $cerebro;
      proxy_set_header Host $http_host;   # required for docker client's sake
      proxy_set_header X-Real-IP $remote_addr; # pass on real client's IP
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_read_timeout 900;
    }
  }`

	// File vhost-utmstack
	vhostUtmStack := `server {
  listen 80 default_server;
  server_name utmclient.utmstack.com;
  return 301 https://utmclient.utmstack.com$request_uri;
}
server {
  listen 443 ssl default_server;
  server_name utmclient.utmstack.com;
  resolver 127.0.0.11 valid=600s ipv6=off;
  set $tomcat http://utm_tomcat:8080;
  ssl_certificate /etc/nginx/single.crt;
  ssl_certificate_key /etc/nginx/single.key;
  ssl_protocols TLSv1.1 TLSv1.2;
  ssl_ciphers 'EECDH+AESGCM:EDH+AESGCM:AES256+EECDH:AES256+EDH';
  ssl_prefer_server_ciphers on;
  ssl_session_cache shared:SSL:10m;
  client_max_body_size 0;
  chunked_transfer_encoding on;
  location / {
    proxy_pass $tomcat;
    proxy_set_header Host $http_host;   # required for docker client's sake
    proxy_set_header X-Real-IP $remote_addr; # pass on real client's IP
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 900;
  }
}`

	// utm_client.sql
	file, err := os.Create("/etc/utmstack/utm_client.sql")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(utmClient)
		fmt.Println("File: '/etc/utmstack/utm_client.sql' ... Ok.")
	}
	file.Close()

	// compose.yml
	file, err = os.Create("/etc/utmstack/compose.yml")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(utmCompose)
		fmt.Println("File: '/etc/utmstack/compose.yml' ... Ok.")
	}
	file.Close()

	// application-prod.yml
	file, err = os.Create("/etc/utmstack/application-prod.yml")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(appProd)
		fmt.Println("File: '/etc/utmstack/application-prod.yml' ... Ok.")
	}
	file.Close()

	// config.json
	file, err = os.Create("/etc/utmstack/config.json")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(probeJSON)
		fmt.Println("File: '/etc/utmstack/config.json' ... Ok.")
	}
	file.Close()

	// vhost-utmstack
	file, err = os.Create("/etc/utmstack/vhost-utmstack")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(vhostUtmStack)
		fmt.Println("File: '/etc/utmstack/vhost-utmstack' ... Ok.")
	}
	file.Close()

	// vhost-elastic
	file, err = os.Create("/etc/utmstack/vhost-elastic")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(vhostElastic)
		fmt.Println("File: '/etc/utmstack/vhost-elastic' ... Ok.")
	}
	file.Close()

	// vhost-cerebro
	file, err = os.Create("/etc/utmstack/vhost-cerebro")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(vhostCerebro)
		fmt.Println("File: '/etc/utmstack/vhost-cerebro' ... Ok.")
	}
	file.Close()

}

func copyLicense() {
	eulaUTM := `Atlas Inside Technology LLC - UTMStack License

By exercising the Licensed Rights (defined below), You accept and agree to be bound by the terms and conditions of this Atlas Inside Technology LLC - UTMStack License ("UTMStack License"). To the extent this UTMStack License may be interpreted as a contract, You are granted the Licensed Rights in consideration of Your acceptance of these terms and conditions, and the Licensor grants You such rights in consideration of benefits the Licensor receives from making the Licensed Material available under these terms and conditions.

# Section 1 – Definitions.

Adapted Material means material subject to Copyright and Similar Rights that is derived from or based upon the Licensed Material and in which the Licensed Material is translated, altered, arranged, transformed, or otherwise modified in a manner requiring permission under the Copyright and Similar Rights held by the Licensor.

Copyright and Similar Rights means copyright and/or similar rights closely related to copyright including, without limitation, performance, broadcast, sound recording, and Sui Generis Database Rights, without regard to how the rights are labeled or categorized. For purposes of this UTMStack License, the rights specified in Section 2 are not Copyright and Similar Rights.

Effective Technological Measures means those measures that, in the absence of proper authority, may not be circumvented under laws fulfilling obligations under Article 11 of the WIPO Copyright Treaty adopted on December 20, 1996, and/or similar international agreements.

Exceptions and Limitations means fair use, fair dealing, and/or any other exception or limitation to Copyright and Similar Rights that applies to Your use of the Licensed Material.

Licensed Material means the source code, compiled software, database, or other material to which the Licensor applied this UTMStack License.

Licensed Rights means the rights granted to You subject to the terms and conditions of this UTMStack License, which are limited to all Copyright and Similar Rights that apply to Your use of the Licensed Material and that the Licensor has authority to license.

Licensor means the individual(s) or entity(ies) granting rights under this UTMStack License.

NonCommercial means not primarily intended for or directed towards commercial advantage or monetary compensation. For purposes of this UTMStack License, the exchange of the Licensed Material for other material subject to Copyright and Similar Rights by digital file-sharing or similar means is NonCommercial provided there is no payment of monetary compensation in connection with the exchange.

Share means to provide material to the public by any means or process that requires permission under the Licensed Rights, such as reproduction, public display, public performance, distribution, dissemination, communication, or importation, and to make material available to the public including in ways that members of the public may access the material from a place and at a time individually chosen by them.

Sui Generis Database Rights means rights other than copyright resulting from Directive 96/9/EC of the European Parliament and of the Council of 11 March 1996 on the legal protection of databases, as amended and/or succeeded, as well as other essentially equivalent rights anywhere in the world.

You means the individual or entity exercising the Licensed Rights under this UTMStack License. Your has a corresponding meaning.

# Section 2 – Scope.

## License grant.

Subject to the terms and conditions of this UTMStack License, the Licensor hereby grants You a worldwide, royalty-free, non-sublicensable, non-exclusive, irrevocable license to exercise the Licensed Rights in the Licensed Material to:

* Use and Share the Licensed Material, in whole or in part, for NonCommercial purposes only.

Exceptions and Limitations. For the avoidance of doubt, where Exceptions and Limitations apply to Your use, this UTMStack License does not apply, and You do not need to comply with its terms and conditions.

## Term.

The term of this UTMStack License is specified in Section 6.

## Downstream recipients.

Offer from the Licensor – Licensed Material. Every recipient of the Licensed Material automatically receives an offer from the Licensor to exercise the Licensed Rights under the terms and conditions of this UTMStack License.

No downstream restrictions. You may not offer or impose any additional or different terms or conditions on, or apply any Effective Technological Measures to, the Licensed Material if doing so restricts exercise of the Licensed Rights by any recipient of the Licensed Material.

No endorsement. Nothing in this UTMStack License constitutes or may be construed as permission to assert or imply that You are, or that Your use of the Licensed Material is, connected with, or sponsored, endorsed, or granted official status by, the Licensor or others designated to receive attribution.
Other rights.

Moral rights, such as the right of integrity, are not licensed under this UTMStack License, nor are publicity, privacy, and/or other similar personality rights; however, to the extent possible, the Licensor waives and/or agrees not to assert any such rights held by the Licensor to the limited extent necessary to allow You to exercise the Licensed Rights, but not otherwise.

Patent and trademark rights are not licensed under this UTMStack License.

To the extent possible, the Licensor waives any right to collect royalties from You for the exercise of the Licensed Rights, whether directly or through a collecting society under any voluntary or waivable statutory or compulsory licensing scheme. In all other cases the Licensor expressly reserves any right to collect such royalties, including when the Licensed Material is used other than for NonCommercial purposes.

# Section 3 – License Conditions.

Your exercise of the Licensed Rights is expressly made subject to the following conditions.

## Attribution.

If You Share the Licensed Material, You must:
retain the following if it is supplied by the Licensor with the Licensed Material:
identification of the creator(s) of the Licensed Material and any others designated to receive attribution, in any reasonable manner requested by the Licensor (including by pseudonym if designated);
a copyright notice;
a notice that refers to this UTMStack License;
a notice that refers to the disclaimer of warranties;
a URI or hyperlink to the Licensed Material to the extent reasonably practicable; and
indicate the Licensed Material is licensed under this UTMStack License, and include the text of, or the URI or hyperlink to, this UTMStack License.

For the avoidance of doubt, You do not have permission under this UTMStack License to make or Share Adapted Material.

You may satisfy the conditions in this section in any reasonable manner based on the medium, means, and context in which You Share the Licensed Material. For example, it may be reasonable to satisfy the conditions by providing a URI or hyperlink to a resource that includes the required information.

If requested by the Licensor, You must remove any of the information required by this section to the extent reasonably practicable.

# Section 4 – Sui Generis Database Rights.

Where the Licensed Rights include Sui Generis Database Rights that apply to Your use of the Licensed Material:
for the avoidance of doubt, Section 2 grants You the right to extract, reuse, reproduce, and Share all or a substantial portion of the contents of the database for NonCommercial purposes only and provided You do not Share Adapted Material;
if You include all or a substantial portion of the database contents in a database in which You have Sui Generis Database Rights, then the database in which You have Sui Generis Database Rights (but not its individual contents) is Adapted Material; and
You must comply with the conditions in Section 3 if You Share all or a substantial portion of the contents of the database.
For the avoidance of doubt, this Section 4 supplements and does not replace Your obligations under this UTMStack License where the Licensed Rights include other Copyright and Similar Rights.

# Section 5 – Disclaimer of Warranties and Limitation of Liability.

Unless otherwise separately undertaken by the Licensor, to the extent possible, the Licensor offers the Licensed Material as-is and as-available, and makes no representations or warranties of any kind concerning the Licensed Material, whether express, implied, statutory, or other. This includes, without limitation, warranties of title, merchantability, fitness for a particular purpose, non-infringement, absence of latent or other defects, accuracy, or the presence or absence of errors, whether or not known or discoverable. Where disclaimers of warranties are not allowed in full or in part, this disclaimer may not apply to You.

To the extent possible, in no event will the Licensor be liable to You on any legal theory (including, without limitation, negligence) or otherwise for any direct, special, indirect, incidental, consequential, punitive, exemplary, or other losses, costs, expenses, or damages arising out of this UTMStack License or use of the Licensed Material, even if the Licensor has been advised of the possibility of such losses, costs, expenses, or damages. Where a limitation of liability is not allowed in full or in part, this limitation may not apply to You.
The disclaimer of warranties and limitation of liability provided above shall be interpreted in a manner that, to the extent possible, most closely approximates an absolute disclaimer and waiver of all liability.

# Section 6 – Term and Termination.

This UTMStack License applies for the term of the Copyright and Similar Rights licensed here. However, if You fail to comply with this UTMStack License, then Your rights under this UTMStack License terminate automatically.

Where Your right to use the Licensed Material has terminated under Section 6, it reinstates:
automatically as of the date the violation is cured, provided it is cured within 30 days of Your discovery of the violation; or
upon express reinstatement by the Licensor.

For the avoidance of doubt, this Section 6 does not affect any right the Licensor may have to seek remedies for Your violations of this UTMStack License.

For the avoidance of doubt, the Licensor may also offer the Licensed Material under separate terms or conditions or stop distributing the Licensed Material at any time; however, doing so will not terminate this UTMStack License.

Sections 1, 5, 6, 7, and 8 survive termination of this UTMStack License.

# Section 7 – Other Terms and Conditions.

The Licensor shall not be bound by any additional or different terms or conditions communicated by You unless expressly agreed.
Any arrangements, understandings, or agreements regarding the Licensed Material not stated herein are separate from and independent of the terms and conditions of this UTMStack License.

# Section 8 – Interpretation.

For the avoidance of doubt, this UTMStack License does not, and shall not be interpreted to, reduce, limit, restrict, or impose conditions on any use of the Licensed Material that could lawfully be made without permission under this UTMStack License.
To the extent possible, if any provision of this UTMStack License is deemed unenforceable, it shall be automatically reformed to the minimum extent necessary to make it enforceable. If the provision cannot be reformed, it shall be severed from this UTMStack License without affecting the enforceability of the remaining terms and conditions.
No term or condition of this UTMStack License will be waived and no failure to comply consented to unless expressly agreed to by the Licensor.
Nothing in this UTMStack License constitutes or may be interpreted as a limitation upon, or waiver of, any privileges and immunities that apply to the Licensor or You, including from the legal processes of any jurisdiction or authority.

# Section 9 - Third Party Materials
UTMStack uses materials from third party licensors.
If you use those materials, you accept the terms expressed in their licenses for the use, modification and distribution of those third-party materials.

## Third Party Materials List

| Material | License |
|---|---|
| psycopg2-binary | LGPL |
| elasticsearch | Apache-2.0 |
| geoip2 | Apache-2.0 |
| grpcio | Apache-2.0 |
| boto3 | Apache-2.0 |
| requests | Apache-2.0 |
| Jinja2 | BSD-3-Clause |
| hug | MIT |
| flask | BSD-3-Clause |
| ldap3 | LGPLv3 |
| pyparsing | MIT |
| python-dateutil | Apache |
| pyasn1 | BSD |
| echart | Apache-2.0 |
| echart-gl | BSD |
| echart-stat | BSD |
| echarts-wordcloud | ISC |
| echartslayer | ISC |
| flat | BSD-3-Clause |
| jsoneditor | Apache-2.0 |
| leaflet | BSD-2-Clause |
| jhipster | Apache-2.0 |
| ngx-drag-drop | BSD-3-Clause |`

	// License-Private-UTMStack.md
	file, err := os.Create("/etc/utmstack/License-Private-UTMStack.md")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(eulaUTM)
		fmt.Println("File: '/etc/utmstack/License-Private-UTMStack.md' ... Ok.")
	}
	file.Close()
}

func selectInstallType() string {
	answer := ""
	for {
		fmt.Println("# -------------------------------------")
		fmt.Println("# [UTMSTACK]: Select Installation Type.")
		fmt.Println("# -------------------------------------")
		fmt.Println("(1) Evaluation Version (free trial up).")
		fmt.Println("(2) Enterprise Version (paid license).")
		fmt.Println("(U)ninstall UTMStack Deployment.")
		fmt.Println("(C)ancel the Setup and do Nothing.")
		fmt.Println("# -------------------------------------")
		fmt.Print("Your answer is?: ")
		fmt.Scanln(&answer)
		opt := string(answer)
		if opt == "1" {
			return "f"
		}
		if opt == "2" {
			return "l"
		}
		if opt == "u" || opt == "U" {
			return "u"
		}
		if opt == "c" || opt == "C" {
			os.Exit(0)
		}
		answer = ""
		fmt.Println("[ERROR] => [UTMSTACK]: (", opt, ") is NOT a valid choice.")
	}
}

func testAccess() { // Testing Product Source access
	fmt.Println("# [UTMSTACK]: Testing Product Source Access ...")
	// curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/utm-stack.zip
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- utm-stack.zip ----------")
	outFront, errFront := exec.Command("sh", "-c", `curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/utm-stack.zip`).Output()
	strFront := string(outFront)
	trimFront := strings.TrimRight(strFront, "\n")
	if trimFront == "" {
		fmt.Println("Error:", curlError(errFront.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimFront)
		fmt.Println("File-Status: utm-stack.zip is Ok.")
	}

	// curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/tomcat.tar.gz
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- tomcat.tar.gz ----------")
	outWar, errWar := exec.Command("sh", "-c", `curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/tomcat.tar.gz`).Output()
	strWar := string(outWar)
	trimWar := strings.TrimRight(strWar, "\n")
	if trimFront == "" {
		fmt.Println("Error:", curlError(errWar.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimWar)
		fmt.Println("File-Status: tomcat.tar.gz is Ok.")
	}

	// curl -k -X GET https://registry.utmstack.com:443/v2/_catalog
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- Docker Registry ----------")
	outReg, errReg := exec.Command("sh", "-c", `curl -k -X GET https://registry.utmstack.com:443/v2/_catalog`).Output()
	strReg := string(outReg)
	trimReg := strings.TrimRight(strReg, "\n")
	if trimReg == "" {
		fmt.Println("Error:", curlError(errReg.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimReg)
		fmt.Println("Repo-Status: Docker Registry is Ok.")
	}

	// Verify each image
	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-bionic/tags/list

	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-cerebro/tags/list

	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-nginx/tags/list

	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-elastic-aws/tags/list

	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-postgres/tags/list

	// curl -k -X GET https://registry.utmstack.com:443/v2/utm-scanner11/tags/list

	// curl -k --head https://updates.utmstack.com/assets/stack
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- Proxy Probe ----------")
	outProbe, errProbe := exec.Command("sh", "-c", `curl -k --head https://updates.utmstack.com/assets/stack`).Output()
	strProbe := string(outProbe)
	trimProbe := strings.TrimRight(strProbe, "\n")
	if trimProbe == "" {
		fmt.Println("Error:", curlError(errProbe.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimProbe)
		fmt.Println("File-Status: Proxy Probe is Ok.")
	}
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: Testing Product Source Access is Done.")
	fmt.Println("")

	// curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/geoip.zip
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- geoip.zip ----------")
	outWar, errWar = exec.Command("sh", "-c", `curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/geoip.zip`).Output()
	strWar = string(outWar)
	trimWar = strings.TrimRight(strWar, "\n")
	if trimFront == "" {
		fmt.Println("Error:", curlError(errWar.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimWar)
		fmt.Println("File-Status: geoip.zip is Ok.")
	}

	// curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/updateUTM
	fmt.Println("")
	fmt.Println("# [UTMSTACK]: ---------- updateUTM ----------")
	outWar, errWar = exec.Command("sh", "-c", `curl -u "upgrade:y#K=[swzk.(DQ7+[w;=#na/958yU$" --head ftp://registry.utmstack.com/updateUTM`).Output()
	strWar = string(outWar)
	trimWar = strings.TrimRight(strWar, "\n")
	if trimFront == "" {
		fmt.Println("Error:", curlError(errWar.Error()))
		fmt.Println("# [UTMSTACK]: Exiting UTMStack Setup ...")
		os.Exit(1)
	} else {
		fmt.Println(trimWar)
		fmt.Println("File-Status: updateUTM is Ok.")
	}

}

func curlError(err string) string {
	trimErr := strings.TrimRight(err, "\n")
	switch trimErr {
	case "exit status 6":
		return "Couldn't resolve host. The given remote host's address was not resolved."
	case "exit status 7":
		return "Wrong port number, wrong host name, wrong protocol or there is network equipment blocking the traffic."
	case "exit status 19":
		return "Got an error from the server when trying to download/access the file."
	case "exit status 51":
		return "The server's SSL/TLS certificate or SSH fingerprint failed verification."
	case "exit status 67":
		return "The user name, password, or similar was not accepted and failed to log in."
	default:
		return err
	}
}

func genCert() {
	// openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /etc/utmstack/single.key -out /etc/utmstack/single.crt -subj '/C=US/ST=Florida/L=Doral/O=UTMStack/OU=Org/CN=utmclient.utmstack.com'
	strCMD := "openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /etc/utmstack/single.key -out /etc/utmstack/single.crt -subj '/C=US/ST=Florida/L=Doral/O=UTMStack/OU=Org/CN=" + domain + "'"
	err := exec.Command("/bin/bash", "-c", strCMD).Run()
	if err != nil {
		fmt.Println("[ERROR]:", err)
	} else {
		fmt.Println("# [UTMSTACK]: Generating self-singed certificate ... Ok.")
	}
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[seededRand.Intn(len(letters))]
	}
	return string(s)
}
