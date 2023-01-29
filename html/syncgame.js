// window.pendingSyncGame = null;

async function setupSyncModal(gameName) {
    await log(`Setting up sync for ${gameName}`);
    await getSyncDryRun(gameName);
    const interval = setInterval(async () => {
        const resultStr = await pollLogs(gameName);
        if (resultStr !== "") {
            const result = JSON.parse(resultStr)
            if (result && result.Finished) {
                clearInterval(interval);

                const messages = result.Message.split("\n");



                log("Finished " + messages.length);
            }    
        }
    }, 500);
}

async function onSyncGameModalClosed(element) {
    const model = document.getElementById("bisync-confirm");
    model.style.display = 'none';
}

async function onSyncGameConfirm(element) {
    const loaderEl = document.getElementById('sync-game-modal-loader');
    loaderEl.style.display = 'block';


}