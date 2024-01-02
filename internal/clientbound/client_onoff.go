package clientbound

import (
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (cm *clientManager) online(end geminio.End) error {
	old, ok := cm.clients.Swap(end.ClientID(), end)
	if ok {
		// if the old connection exits, offline it
		oldend := old.(geminio.End)
		if err := oldend.Close(); err != nil {
			klog.Warningf("kick off old end err: %s, clientID: %s", err, end.ClientID())
		}
	}
	return nil
}
