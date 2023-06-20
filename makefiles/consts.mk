# Copyright 2022 Charlie Chiang
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Those variables assigned with ?= can be overridden by setting them
# manually on the command line or using environment variables.

# Go version used as the image of the build container, grabbed from go.mod
GO_VERSION       := $(shell grep -E '^go [[:digit:]]{1,3}\.[[:digit:]]{1,3}$$' go.mod | sed 's/go //')
# Local Go release version (only supports go1.16 and later)
LOCAL_GO_VERSION := $(shell go env GOVERSION 2>/dev/null | grep -oE "go[[:digit:]]{1,3}\.[[:digit:]]{1,3}" || echo "none")

# Warn if local go release version is different from what is specified in go.mod.
ifneq (none, $(LOCAL_GO_VERSION))
  ifneq (go$(GO_VERSION), $(LOCAL_GO_VERSION))
    $(warning Your local Go release ($(LOCAL_GO_VERSION)) is different from the one that this go module assumes (go$(GO_VERSION)).)
  endif
endif

# Set DEBUG to 1 to optimize binary for debugging, otherwise for release
DEBUG ?=

# Version string, use git tag by default
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "UNKNOWN")
GIT_COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo "UNKNOWN")

GOOS        ?=
GOARCH      ?=
# If user has not defined GOOS/GOARCH, use Go defaults.
# If user don't have Go, use the os/arch of their machine.
ifeq (, $(shell which go))
  HOSTOS     := $(shell uname -s | tr '[:upper:]' '[:lower:]')
  HOSTARCH   := $(shell uname -m)
  ifeq ($(HOSTARCH),x86_64)
    HOSTARCH := amd64
  endif
  OS         := $(if $(GOOS),$(GOOS),$(HOSTOS))
  ARCH       := $(if $(GOARCH),$(GOARCH),$(HOSTARCH))
else
  OS         := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
  ARCH       := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
endif

# Binary name
BIN_BASENAME      := $(BIN)
# Binary name with extended info, i.e. version-os-arch
BIN_FULLNAME      := $(BIN)-$(VERSION)-$(OS)-$(ARCH)
# Package filename (generated by `make package'). Use zip for Windows, tar.gz for all other platforms.
PKG_FULLNAME      := $(BIN_FULLNAME).tar.gz
# Checksum filename
CHECKSUM_FULLNAME := $(BIN)-$(VERSION)-checksums.txt

# This holds build output and helper tools
DIST           := bin
# Full output directory
BIN_OUTPUT_DIR := $(DIST)/$(BIN)-$(VERSION)
PKG_OUTPUT_DIR := $(BIN_OUTPUT_DIR)/packages
# Full output path with filename
OUTPUT         := $(BIN_OUTPUT_DIR)/$(BIN_FULLNAME)
PKG_OUTPUT     := $(PKG_OUTPUT_DIR)/$(PKG_FULLNAME)
