package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distatus/battery"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func getConfig(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, config)
}

func setConfig(c *gin.Context) {
	var cfg Config
	if err := c.BindJSON(&cfg); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if cfg.Limit < 10 || cfg.Limit > 100 {
		err := fmt.Errorf("limit must be between 10 and 100, got %d", cfg.Limit)
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	config = cfg
	if err := saveConfig(); err != nil {
		logrus.Errorf("saveConfig failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("set config: %#v", cfg)

	// Immediate single maintain loop, to avoid waiting for the next loop
	maintainLoop()
	c.IndentedJSON(http.StatusCreated, "ok")
}

func getLimit(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, config.Limit)
}

func setLimit(c *gin.Context) {
	var l int
	if err := c.BindJSON(&l); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if l < 10 || l > 100 {
		err := fmt.Errorf("limit must be between 10 and 100, got %d", l)
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	config.Limit = l
	if err := saveConfig(); err != nil {
		logrus.Errorf("saveConfig failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("set charging limit to %d", l)

	var msg string
	charge, err := smcConn.GetBatteryCharge()
	if err != nil {
		msg = fmt.Sprintf("set charging limit to %d", l)
	} else {
		msg = fmt.Sprintf("set charging limit to %d, current charge: %d", l, charge)
		if charge > config.Limit {
			msg += ", you may need to drain your battery below the limit to see any effect"
		}
	}

	// Immediate single maintain loop, to avoid waiting for the next loop
	maintainLoop()

	c.IndentedJSON(http.StatusCreated, msg)
}

func setPreventIdleSleep(c *gin.Context) {
	var p bool
	if err := c.BindJSON(&p); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	config.PreventIdleSleep = p
	if err := saveConfig(); err != nil {
		logrus.Errorf("saveConfig failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("set prevent idle sleep to %t", p)

	c.IndentedJSON(http.StatusCreated, "ok")
}

func setDisableChargingPreSleep(c *gin.Context) {
	var d bool
	if err := c.BindJSON(&d); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	config.DisableChargingPreSleep = d
	if err := saveConfig(); err != nil {
		logrus.Errorf("saveConfig failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("set disable charging pre sleep to %t", d)

	c.IndentedJSON(http.StatusCreated, "ok")
}

func setAdapter(c *gin.Context) {
	var d bool
	if err := c.BindJSON(&d); err != nil {
		c.IndentedJSON(http.StatusBadRequest, err.Error())
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if d {
		if err := smcConn.EnableAdapter(); err != nil {
			logrus.Errorf("enablePowerAdapter failed: %v", err)
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		logrus.Infof("enabled power adapter")
	} else {
		if err := smcConn.DisableAdapter(); err != nil {
			logrus.Errorf("disablePowerAdapter failed: %v", err)
			c.IndentedJSON(http.StatusInternalServerError, err.Error())
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		logrus.Infof("disabled power adapter")
	}

	c.IndentedJSON(http.StatusCreated, "ok")
}

func getAdapter(c *gin.Context) {
	enabled, err := smcConn.IsAdapterEnabled()
	if err != nil {
		logrus.Errorf("getAdapter failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, enabled)
}

func getCharging(c *gin.Context) {
	charging, err := smcConn.IsChargingEnabled()
	if err != nil {
		logrus.Errorf("getCharging failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, charging)
}

func getBatteryInfo(c *gin.Context) {
	batteries, err := battery.GetAll()
	if err != nil {
		logrus.Errorf("getBatteryInfo failed: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, err.Error())
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if len(batteries) == 0 {
		logrus.Errorf("no batteries found")
		c.IndentedJSON(http.StatusInternalServerError, "no batteries found")
		_ = c.AbortWithError(http.StatusInternalServerError, errors.New("no batteries found"))
		return
	}

	bat := batteries[0] // All Apple Silicon MacBooks only have one battery. No need to support more.
	if bat.State == battery.Discharging {
		bat.ChargeRate = -bat.ChargeRate
	}

	c.IndentedJSON(http.StatusOK, bat)
}
