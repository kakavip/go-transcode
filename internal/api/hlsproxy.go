package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"vimai/ads-transcode/hlsproxy"
)

const hlsProxyPerfix = "/hlsproxy/"

var hlsProxyManagers map[string]hlsproxy.Manager = make(map[string]hlsproxy.Manager)

func (a *ApiManagerCtx) HLSProxy(r chi.Router) {
	r.Get(hlsProxyPerfix+"{sourceId}/*", func(w http.ResponseWriter, r *http.Request) {
		ID := chi.URLParam(r, "sourceId")

		// check if stream exists
		baseUrl, ok := a.config.HlsProxy[ID]
		if !ok {
			http.Error(w, "404 hls proxy source not found", http.StatusNotFound)
			return
		}

		manager, ok := hlsProxyManagers[ID]
		if !ok {
			// create new manager
			manager = hlsproxy.New(baseUrl, hlsProxyPerfix+ID+"/")
			hlsProxyManagers[ID] = manager
		}

		// if this is playlist request
		if strings.HasSuffix(r.URL.String(), ".m3u8") {
			manager.ServePlaylist(w, r)
		} else {
			manager.ServeMedia(w, r)
		}
	})
}
