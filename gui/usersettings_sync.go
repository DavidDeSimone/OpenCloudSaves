package gui

import (
	"context"
	"fmt"
	"io/fs"
	"opencloudsave/core"
)

func showDefinitionSyncScreen(dm core.GameDefManager) error {
	w := GetRootWindow()

	pendingUserSettingContext, cancelFunc := context.WithCancel(context.Background())

	w.Bind("cancelPendingGamedefSync", func() {
		cancelFunc()
		w.Dispatch(func() {
			w.Eval("refresh()")
		})
	})

	go (func() {
		core.GetUserSettingsManager().RequestSync(pendingUserSettingContext, dm.GetUserOverrideLocation())
		if pendingUserSettingContext.Err() == context.Canceled {
			core.InfoLogger.Println("User Cancelled Gamedef Sync")
			return
		}

		dm.ApplyUserOverrides()
		w.Dispatch(func() {
			w.Eval("refresh()")
		})
	})()

	cssContent, err := fs.ReadFile(html, "html/usersettings.css")
	if err != nil {
		return err
	}

	css := fmt.Sprintf("<style>%v</style>\n", string(cssContent))

	result, err := fs.ReadFile(html, "html/usersettings_sync.html")
	if err != nil {
		return err
	}

	w.SetHtml(css + string(result))
	return nil
}
