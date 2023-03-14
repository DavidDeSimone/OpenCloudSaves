const CurrentSyncState = {
    // If there is an active rclone sync operation pending.
    hasActiveSyncOperation: false,
    // The name of the game represented by the sync system.
    gameToSync: null,
    retryDryRun: false,
};

function recordSyncMessage(message) {
    const multisync = document.getElementById('bisync-line-cont');
    const lineDiv = document.createElement('div');
    lineDiv.className = "bisync-line";
    lineDiv.innerText = message;
    multisync.appendChild(lineDiv);
}

async function onFinished(result, dryRun) {
    log("Finished Sync");
    const messages = (result && result.Message) ? result.Message.split("\n") : [];
    const lineContEl = document.getElementById('bisync-line-cont');
    const loaderEl = document.getElementById('sync-game-modal-loader');
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    if (dryRun) {
        bisyncSubtitle.innerText = "Your data is not sync'd yet, please read the summary and conduct the sync if everything looks good.";
    } else {
        bisyncSubtitle.innerText = "Sync Complete";
    }

    loaderEl.style.display = 'none';
    lineContEl.style.display = 'block';
    syncConfirm.disabled = false;
    syncCancel.disabled = false;
    for (let i = 0; i < messages.length; ++i) {
        if (messages[i] === null || messages[i] === "") {
            continue;
        }

        recordSyncMessage(messages[i]);
    }
}

async function onSyncError(error, dryRun) {
    log("Error " + error);
    const lineContEl = document.getElementById('bisync-line-cont');
    const loaderEl = document.getElementById('sync-game-modal-loader');
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    bisyncSubtitle.innerText = "Error while syncing!";

    loaderEl.style.display = 'none';
    lineContEl.style.display = 'block';
    syncConfirm.disabled = false;
    syncCancel.disabled = false;

    const lineDiv = document.createElement("div");
    lineDiv.className = "bisync-line";
    if (!dryRun) {
        lineDiv.innerText = `Error while performing sync: ${error}`;
    } else {
        lineDiv.innerText = `Error while performing dry-run of sync: ${error}`;
    }

    lineContEl.appendChild(lineDiv);

    syncConfirm.innerText = "Retry";
    CurrentSyncState.retryDryRun = dryRun;

    syncConfirm.style.display = 'block';
    syncCancel.style.display = 'block';
} 

function resetSyncModal() {
    const lineContEl = document.getElementById('bisync-line-cont');
    const loaderEl = document.getElementById('sync-game-modal-loader');
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    bisyncSubtitle.innerText = "Your data is not sync'd yet, please read the summary and conduct the sync if everything looks good.";
    lineContEl.innerHTML = "";
    syncConfirm.disabled = true;
    syncCancel.disabled = true;
    syncCancel.style.display = 'block';
    syncConfirm.style.display = 'block';
    loaderEl.style.display = 'block';
    CurrentSyncState.gameToSync = null;
    CurrentSyncState.retryDryRun = false;
    CurrentSyncState.hasActiveSyncOperation = false;
    syncConfirm.innerText = "Confirm";
}

async function pollIteration(gameName, dryRun) {
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');

    let error = null;
    const resultStr = await pollLogs(gameName)
                .catch(e => {
                    onSyncError(e, dryRun);
                    error = e;
                });

    if (error !== null) {
        return true;
    }

    if (resultStr === "") {
        return false;
    }

    const result = JSON.parse(resultStr);
    if (result && result.Finished) {
        await onFinished(result, dryRun)
            .catch(e => log(`Error: ${e}`));
        if (dryRun) {
            syncConfirm.style.display = 'block';
            syncCancel.style.display = 'block';    
        }

        return true;
    }

    recordSyncMessage(result.Message);
    return false;
}

async function pollLoopForSyncGame(gameName, dryRun) {
    const timerValue = 250;
    return new Promise(async (resolve) => {
        const func = async () => {
            const complete = await pollIteration(gameName, dryRun);

            if (complete) {
                resolve();
            } else {
                setTimeout(func, timerValue);
            }
        }

        setTimeout(func, timerValue);
    });
}

async function sync(gameName, dryRun) {
    log(`Checking If should perform dry run - ${dryRun}`);
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');
    syncConfirm.style.display = 'none';
    syncCancel.style.display = 'none';

    recordSyncMessage(`---------------------------------------------------------`);
    recordSyncMessage(`Syncing: ${gameName}`);
    recordSyncMessage(`---------------------------------------------------------`);

    if (dryRun) {
        bisyncSubtitle.innerText = "Simulating transfer - please wait";
        await getSyncDryRun(gameName);
    } else {
        bisyncSubtitle.innerText = "Performing sync - please wait";
        await syncGame(gameName);
    }

    CurrentSyncState.hasActiveSyncOperation = true;
    await pollLoopForSyncGame(gameName, dryRun);
    CurrentSyncState.hasActiveSyncOperation = false;

    recordSyncMessage(`---------------------------------------------------------`);
    recordSyncMessage(`Sync Complete: ${gameName}`);
    recordSyncMessage(`---------------------------------------------------------`);                     
}

async function setupSyncModal(gameName) {
    log(`Setting up sync for ${gameName}`);
    resetSyncModal();
    document.getElementById('accordion-cont').style.display='none';

    CurrentSyncState.gameToSync = gameName;
    const shouldPerformDryRun = await getShouldPerformDryRun();
    sync(gameName, shouldPerformDryRun);
}

async function confirmCancellation() {
    makeConfirmationPopup({
        title: `Are you sure you want to cancel sync'ing ${CurrentSyncState.gameToSync}?`,
        subtitle: `Hitting confirm will cancel your pending sync operation.`,
        onConfirm: async () => {
            await cancelPendingSync(CurrentSyncState.gameToSync);
            CurrentSyncState.gameToSync = null;
            CurrentSyncState.hasActiveSyncOperation = false;
            refresh();
        },
    });
}

async function onSyncGameModalClosed(element) {
    if (CurrentSyncState.hasActiveSyncOperation) {
        await confirmCancellation();
        return;
    }

    const model = document.getElementById("bisync-confirm");
    model.style.display = 'none';
    document.getElementById('accordion-cont').style.display='block';
    refresh();
}

async function onSyncGameConfirm(element) {
    const loaderEl = document.getElementById('sync-game-modal-loader');
    const lineContEl = document.getElementById('bisync-line-cont');

    loaderEl.style.display = 'block';
    lineContEl.innerHTML = "";
    const dryRun = CurrentSyncState.retryDryRun;
    CurrentSyncState.retryDryRun = false;
    await sync(CurrentSyncState.gameToSync, dryRun);
}