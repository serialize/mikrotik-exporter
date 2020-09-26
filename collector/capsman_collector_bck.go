package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type capsMan2Collector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newCapsMan2Collector() routerOSCollector {
	c := &capsMan2Collector{}
	c.init()
	return c
}

func (c *capsMan2Collector) init() {
	c.props = []string{"interface", "mac-address", "radio-name", "ssid", "uptime", "rx-rate", "rx-signal", "tx-rate", "tx-signal", "packets", "bytes"}
	labelNames := []string{"name", "address", "interface", "mac_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[:len(c.props)-3] {
		c.descriptions[p] = descriptionForPropertyName("caps_man", p, labelNames)
	}
}

func (c *capsMan2Collector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *capsMan2Collector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *capsMan2Collector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/caps-man/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching capsman metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *capsMan2Collector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]

	for _, p := range c.props[2 : len(c.props)-3] {
		c.collectMetricForProperty(p, iface, mac, re, ctx)
	}
}

func (c *capsMan2Collector) collectMetricForProperty(property, iface, mac string, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing capsman metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, iface, mac)
}
