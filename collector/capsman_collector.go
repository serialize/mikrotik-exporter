package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type capsManCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newCapsManCollector() routerOSCollector {
	c := &capsManCollector{}
	c.init()
	return c
}

func (c *capsManCollector) init() {
	c.props = []string{"interface", "mac-address", "ssid", "tx-rate", "tx-signal", "rx-rate", "rx-signal", "packets", "bytes"}
	labelNames := []string{"name", "address", "interface", "mac_address", "ssid"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[:len(c.props)-2] {
		c.descriptions[p] = descriptionForPropertyName("capsman", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.descriptions["tx_"+p] = descriptionForPropertyName("capsman", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("capsman", "rx_"+p, labelNames)
	}
}

func (c *capsManCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *capsManCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *capsManCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/caps-man/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching caps man metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *capsManCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]
	ssid := re.Map["ssid"]
	for _, p := range c.props[5 : len(c.props)-2] {
		c.collectMetricForProperty(p, iface, mac, ssid, re, ctx)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.collectMetricForTXRXCounters(p, iface, mac, ssid, re, ctx)
	}
}

func (c *capsManCollector) collectMetricForProperty(property, iface, mac string, ssid string, re *proto.Sentence, ctx *collectorContext) {
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
		}).Error("error parsing caps man metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
}

func (c *capsManCollector) collectMetricForTXRXCounters(property, iface, mac string, ssid string, re *proto.Sentence, ctx *collectorContext) {
	tx, rx, err := splitStringToFloats(re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing caps man metric value")
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
	ctx.ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, ctx.device.Name, ctx.device.Address, iface, mac, ssid)
}
