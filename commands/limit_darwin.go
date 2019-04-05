// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"golang.org/x/sys/unix"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var _ cmder = (*limitCmd)(nil)

type limitCmd struct {
	*baseCmd
}

func newLimitCmd() *limitCmd {
	ccmd := &cobra.Command{
		Use:   "ulimit",
		Short: "Check system ulimit settings",
		Long: `Hugo will inspect the current ulimit settings on the system.
This is primarily to ensure that Hugo can watch enough files on some OSs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var rLimit unix.Rlimit
			err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				return newSystemError("Error Getting Rlimit ", err)
			}

			jww.FEEDBACK.Println("Current rLimit:", rLimit)

			jww.FEEDBACK.Println("Attempting to increase limit")
			rLimit.Max = 999999
			rLimit.Cur = 999999
			err = unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				return newSystemError("Error Setting rLimit ", err)
			}
			err = unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				return newSystemError("Error Getting rLimit ", err)
			}
			jww.FEEDBACK.Println("rLimit after change:", rLimit)

			return nil
		},
	}

	return &limitCmd{baseCmd: newBaseCmd(ccmd)}
}

func tweakLimit() {
	var rLimit unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		jww.ERROR.Println("Unable to obtain rLimit", err)
	}
	if true || rLimit.Cur < rLimit.Max {
		rLimit.Max = 64000
		rLimit.Cur = 64000
		err = unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			jww.WARN.Println("Unable to increase number of open files limit", err)
		}
	}
}
