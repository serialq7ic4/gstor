package benchmark

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type DiskTarget struct {
	Name          string `json:"name"`
	DevicePath    string `json:"device_path"`
	DeviceByID    string `json:"device_by_id,omitempty"`
	SerialNumber  string `json:"serial_number,omitempty"`
	Vendor        string `json:"vendor,omitempty"`
	Model         string `json:"model,omitempty"`
	Firmware      string `json:"firmware,omitempty"`
	CapacityBytes uint64 `json:"capacity_bytes,omitempty"`
	Capacity      string `json:"capacity,omitempty"`
	MediaType     string `json:"media_type"`
	InterfaceType string `json:"interface_type"`
}

type Profile struct {
	Name              string           `json:"name"`
	Version           string           `json:"version"`
	Hash              string           `json:"hash"`
	BaselineCandidate bool             `json:"baseline_candidate"`
	Cases             []CaseDefinition `json:"cases"`
}

type CaseDefinition struct {
	ID        string `json:"case_id"`
	Order     int    `json:"case_order"`
	Phase     string `json:"phase"`
	RW        string `json:"rw"`
	BlockSize string `json:"block_size"`
	RWMixRead int    `json:"rwmixread,omitempty"`
	Offset    string `json:"offset,omitempty"`
	Size      string `json:"size,omitempty"`
}

type Pressure struct {
	NumJobs int `json:"numjobs"`
	IODepth int `json:"iodepth"`
}

func SelectProfile(name string) (Profile, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default", "default_v1":
		return withProfileHash(Profile{
			Name:              "default",
			Version:           "v1",
			BaselineCandidate: true,
			Cases:             defaultCases(),
		}), nil
	case "short", "short_v1":
		return withProfileHash(Profile{
			Name:              "short",
			Version:           "v1",
			BaselineCandidate: false,
			Cases:             shortCases(),
		}), nil
	default:
		return Profile{}, fmt.Errorf("unsupported benchmark profile %q", name)
	}
}

func withProfileHash(profile Profile) Profile {
	hashInput := struct {
		Name              string           `json:"name"`
		Version           string           `json:"version"`
		BaselineCandidate bool             `json:"baseline_candidate"`
		Cases             []CaseDefinition `json:"cases"`
	}{
		Name:              profile.Name,
		Version:           profile.Version,
		BaselineCandidate: profile.BaselineCandidate,
		Cases:             profile.Cases,
	}
	data, err := json.Marshal(hashInput)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	profile.Hash = hex.EncodeToString(sum[:])
	return profile
}

func defaultCases() []CaseDefinition {
	var cases []CaseDefinition
	order := 1
	add := func(phase string, rw string, blockSize string, rwmixread int) {
		cases = append(cases, CaseDefinition{
			ID:        fmt.Sprintf("%s_%s_%s", phase, rw, blockSize),
			Order:     order,
			Phase:     phase,
			RW:        rw,
			BlockSize: blockSize,
			RWMixRead: rwmixread,
		})
		order++
	}

	add("latency_probe", "randread", "4k", 0)
	add("latency_probe", "randwrite", "4k", 0)

	for _, bs := range []string{"4k", "8k", "16k", "32k", "64k"} {
		add("random_perf", "randread", bs, 0)
	}
	for _, bs := range []string{"4k", "8k", "16k", "32k", "64k"} {
		add("random_perf", "randwrite", bs, 0)
	}
	for _, bs := range []string{"4k", "16k", "64k"} {
		add("mixed_random", "randrw", bs, 70)
	}
	for _, bs := range []string{"128k", "256k", "512k", "1M", "4M"} {
		add("sequential_perf", "read", bs, 0)
	}
	for _, bs := range []string{"128k", "256k", "512k", "1M", "4M"} {
		add("sequential_perf", "write", bs, 0)
	}

	for i := range cases {
		if cases[i].Phase == "sequential_perf" {
			cases[i].Offset = "40%"
			cases[i].Size = "20%"
		}
	}

	return cases
}

func shortCases() []CaseDefinition {
	defaults := defaultCases()
	ids := map[string]bool{
		"latency_probe_randread_4k":  true,
		"latency_probe_randwrite_4k": true,
		"random_perf_randread_4k":    true,
		"random_perf_randwrite_4k":   true,
		"mixed_random_randrw_16k":    true,
		"sequential_perf_read_1M":    true,
		"sequential_perf_write_1M":   true,
	}

	cases := make([]CaseDefinition, 0, len(ids))
	for _, c := range defaults {
		if ids[c.ID] {
			c.Order = len(cases) + 1
			cases = append(cases, c)
		}
	}
	return cases
}

func PressureFor(target DiskTarget, phase string) Pressure {
	mediaType := strings.ToUpper(strings.TrimSpace(target.MediaType))
	interfaceType := strings.ToUpper(strings.TrimSpace(target.InterfaceType))

	if phase == "latency_probe" {
		return Pressure{NumJobs: 1, IODepth: 1}
	}

	if interfaceType == "NVME" {
		switch phase {
		case "sequential_perf":
			return Pressure{NumJobs: 2, IODepth: 32}
		default:
			return Pressure{NumJobs: 4, IODepth: 32}
		}
	}

	if mediaType == "SSD" {
		switch phase {
		case "mixed_random", "sequential_perf":
			return Pressure{NumJobs: 1, IODepth: 16}
		default:
			return Pressure{NumJobs: 1, IODepth: 32}
		}
	}

	switch phase {
	case "random_perf", "mixed_random":
		return Pressure{NumJobs: 1, IODepth: 4}
	default:
		return Pressure{NumJobs: 1, IODepth: 1}
	}
}
