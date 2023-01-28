window.pendingSyncGame = null;

async function onSyncGameModalClosed(element) {
    await log("Value: " + window.pendingSyncGame)
    const model = document.getElementById("bisync-confirm");
    model.style.display = 'none';
}

async function onSyncGameConfirm(element) {
    await log(`Syncing Game ${window.pendingSyncGame}`);
    const loaderEl = document.getElementById('sync-game-modal-loader');
    loaderEl.style.display = 'block';


}