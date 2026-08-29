package v1

import (
	"net/http"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
)

// HandlePluginsGet godoc
//
//	@Summary	List installed plugins and their capabilities
//	@Tags		Plugins
//	@Produce	json
//	@Success	200	{array}	plugins.PluginInfo
//	@Router		/v1/plugins [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginsGet() errchain.HandlerFunc {
	fn := func(r *http.Request) ([]plugins.PluginInfo, error) {
		if ctrl.svc.Plugins == nil {
			return []plugins.PluginInfo{}, nil
		}
		return ctrl.svc.Plugins.GetPluginInfos(), nil
	}

	return adapters.Command(fn, http.StatusOK)
}
