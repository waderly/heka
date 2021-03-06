/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is the Mozilla Foundation.
# Portions created by the Initial Developer are Copyright (C) 2012
# the Initial Developer. All Rights Reserved.
#
# Contributor(s):
#   Victor Ng (vng@mozilla.com)
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package http

import (
	"code.google.com/p/gomock/gomock"
	. "github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/pipeline/testsupport"
	"github.com/mozilla-services/heka/pipelinemock"
	plugins_ts "github.com/mozilla-services/heka/plugins/testsupport"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"runtime"
	"time"
)

func HttpInputSpec(c gs.Context) {
	t := &pipeline_ts.SimpleT{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pConfig := NewPipelineConfig(nil)

	json_post := `{"uuid": "xxBI3zyeXU+spG8Uiveumw==", "timestamp": 1372966886023588, "hostname": "Victors-MacBook-Air.local", "pid": 40183, "fields": [{"representation": "", "value_type": "STRING", "name": "cef_meta.syslog_priority", "value_string": [""]}, {"representation": "", "value_type": "STRING", "name": "cef_meta.syslog_ident", "value_string": [""]}, {"representation": "", "value_type": "STRING", "name": "cef_meta.syslog_facility", "value_string": [""]}, {"representation": "", "value_type": "STRING", "name": "cef_meta.syslog_options", "value_string": [""]}], "logger": "", "env_version": "0.8", "type": "cef", "payload": "Jul 04 15:41:26 Victors-MacBook-Air.local CEF:0|mozilla|weave|3|xx\\\\|x|xx\\\\|x|5|cs1Label=requestClientApplication cs1=MySuperBrowser requestMethod=GET request=/ src=127.0.0.1 dest=127.0.0.1 suser=none", "severity": 6}'`

	c.Specify("A HttpInput", func() {

		httpInput := HttpInput{}
		ith := new(plugins_ts.InputTestHelper)
		ith.MockHelper = pipelinemock.NewMockPluginHelper(ctrl)
		ith.MockInputRunner = pipelinemock.NewMockInputRunner(ctrl)

		c.Specify("honors time ticker to flush", func() {

			startInput := func() {
				go func() {
					err := httpInput.Run(ith.MockInputRunner, ith.MockHelper)
					c.Expect(err, gs.IsNil)
				}()
			}

			ith.Pack = NewPipelinePack(pConfig.InputRecycleChan())
			ith.PackSupply = make(chan *PipelinePack, 1)
			ith.PackSupply <- ith.Pack

			// Spin up a http server
			server, err := plugins_ts.NewOneHttpServer(json_post, "localhost", 9876)
			c.Expect(err, gs.IsNil)
			go server.Start("/")
			time.Sleep(10 * time.Millisecond)

			config := httpInput.ConfigStruct().(*HttpInputConfig)
			decoderName := "PayloadJsonDecoder"
			config.DecoderName = decoderName
			config.Url = "http://localhost:9876/"
			tickChan := make(chan time.Time)

			ith.MockInputRunner.EXPECT().LogMessage(gomock.Any()).Times(2)

			ith.MockHelper.EXPECT().PipelineConfig().Return(pConfig)
			ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
			ith.MockInputRunner.EXPECT().Ticker().Return(tickChan)

			mockDecoderRunner := pipelinemock.NewMockDecoderRunner(ctrl)

			// Stub out the DecoderRunner input channel so that we can
			// inspect bytes later on
			dRunnerInChan := make(chan *PipelinePack, 1)
			mockDecoderRunner.EXPECT().InChan().Return(dRunnerInChan)

			ith.MockHelper.EXPECT().DecoderRunner(decoderName).Return(mockDecoderRunner, true)

			err = httpInput.Init(config)
			c.Assume(err, gs.IsNil)
			startInput()

			tickChan <- time.Now()

			// We need for the pipeline to finish up
			time.Sleep(50 * time.Millisecond)
		})

		c.Specify("short circuits packs into the router", func() {

			startInput := func() {
				go func() {
					err := httpInput.Run(ith.MockInputRunner, ith.MockHelper)
					c.Expect(err, gs.IsNil)
				}()
			}
			ith.Pack = NewPipelinePack(pConfig.InputRecycleChan())
			ith.PackSupply = make(chan *PipelinePack, 1)
			ith.PackSupply <- ith.Pack

			config := httpInput.ConfigStruct().(*HttpInputConfig)
			config.Url = "http://localhost:9876/"
			tickChan := make(chan time.Time)

			ith.MockInputRunner.EXPECT().LogMessage(gomock.Any()).Times(2)

			ith.MockHelper.EXPECT().PipelineConfig().Return(pConfig)
			ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
			ith.MockInputRunner.EXPECT().Ticker().Return(tickChan)

			err := httpInput.Init(config)
			c.Assume(err, gs.IsNil)
			startInput()

			tickChan <- time.Now()

			// We need for the pipeline to finish up
			time.Sleep(50 * time.Millisecond)
		})

		ith.MockInputRunner.EXPECT().LogMessage(gomock.Any())
		httpInput.Stop()
		runtime.Gosched() // Yield so the stop can happen before we return.

	})
}
