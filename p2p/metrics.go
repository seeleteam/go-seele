/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import metrics "github.com/rcrowley/go-metrics"

var (
	metricsAddPeerMeter    = metrics.NewRegisteredMeter("p2p.addpeer", nil)
	metricsDeletePeerMeter = metrics.NewRegisteredMeter("p2p.deletepeer", nil)
	metricsPeerCountGauge  = metrics.NewRegisteredGauge("p2p.peercount", nil)
)
