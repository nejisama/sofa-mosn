package mosn

import (
	"sync"
	"testing"

	"github.com/urfave/cli"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
)

func TestStageManager(t *testing.T) {
	stm := NewStageManager(&cli.Context{}, "")
	// test for mock
	testCall := 0
	configmanager.RegisterConfigLoadFunc(func(p string) *v2.MOSNConfig {
		if testCall != 2 {
			t.Errorf("load config is not called after 2 stages registered")
		}
		testCall = 0
		return &v2.MOSNConfig{}
	})
	defer configmanager.RegisterConfigLoadFunc(configmanager.DefaultConfigLoad)
	stm.newMosn = func(*v2.MOSNConfig) *Mosn {
		testCall = 0
		return &Mosn{
			wg: sync.WaitGroup{},
		}
	}

	stm.AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 1 {
			t.Errorf("unexpected params parsed stage call: 1")
		}
	}).AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 2 {
			t.Errorf("unexpected params parsed stage call: 2")
		}
	}).AppendInitStage(func(_ *v2.MOSNConfig) {
		testCall++
		if testCall != 1 {
			t.Errorf("unexpected init stage call: 1")
		}
	}).AppendPreStartStage(func(_ *Mosn) {
		testCall++
		if testCall != 1 {
			t.Errorf("pre start stage call: 1")
		}
	}).AppendStartStage(func(_ *Mosn) {
		testCall++
		if testCall != 2 {
			t.Errorf("pre start stage call: 2")
		}
	})
	if testCall != 0 {
		t.Errorf("should call nothing")
	}
	stm.Run()
	if !(testCall == 2 &&
		stm.data.mosn != nil &&
		stm.data.config != nil) {
		t.Errorf("stage manager runs failed...")
	}
}
