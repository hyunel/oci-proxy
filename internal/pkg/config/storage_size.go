package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type StorageSize int64

func (s *StorageSize) Bytes() int64 {
	return int64(*s)
}

func (s *StorageSize) UnmarshalYAML(value *yaml.Node) error {
	var sizeStr string
	if err := value.Decode(&sizeStr); err != nil {
		return err
	}

	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	if sizeStr == "" {
		*s = 0
		return nil
	}

	sizeStr = strings.TrimSuffix(sizeStr, "B")

	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"T", 1024 * 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"M", 1024 * 1024},
		{"K", 1024},
	}

	for _, unit := range units {
		if strings.HasSuffix(sizeStr, unit.suffix) {
			valueStr := strings.TrimSuffix(sizeStr, unit.suffix)
			parsedValue, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return fmt.Errorf("invalid size value: %s", valueStr)
			}
			*s = StorageSize(parsedValue * float64(unit.multiplier))
			return nil
		}
	}

	parsedValue, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return fmt.Errorf("invalid size value: %s", sizeStr)
	}
	*s = StorageSize(parsedValue)
	return nil
}
