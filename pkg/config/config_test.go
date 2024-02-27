package config

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		want    Configuration
		wantErr bool
	}{
		{
			name: "tryrun",
			want: Configuration{
				Daemon: Daemon{
					RLimit: RLimit{
						NumFile: -1,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("ParseFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	conf := &Configuration{
		Daemon: Daemon{
			RLimit: RLimit{
				NumFile: 1024,
			},
			PProf: PProf{
				Addr: "0.0.0.0:6060",
			},
		},
		Edgebound:    Edgebound{},
		Servicebound: Servicebound{},
	}
	_, err := yaml.Marshal(conf)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestGenDefaultConfig(t *testing.T) {
	file, err := os.OpenFile("./config.yaml", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	err = genDefaultConfig(file)
	if err != nil {
		t.Error(err)
	}
}
