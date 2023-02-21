// window.pendingSyncGame = null;
let pendingSyncGame = null;
let retryDryRun = false;

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


    let hasSeenOperations = false;
    for (let i = 0; i < messages.length; ++i) {
        if (messages[i] === null || messages[i] === "") {
            continue;
        }

        let result = null;
        try {
            result = JSON.parse(messages[i]);
        } catch (e) {
            log(`Error in message processing ${e}`);
            continue;
        }

        // We strip this out for end users - bisync is good enough
        // for our application, and we will warn on our main page prior 
        // to usage
        if (result.msg.startsWith("bisync is EXPERIMENTAL")) {
            continue;
        }

        const lineDiv = document.createElement("div");
        lineDiv.className = "bisync-line";

        // With current rclone format, we want to present all output lines
        // up until the last pending operation. The foramt of the result is 
        // STATEMENT OF OPERATION (COPY FOLDER1 TO FOLDER2)
        // OPERATIONS
        // Statement of success

        if (dryRun) {
            // That statement may be confusing to the end users, so we will omit
            if (result.object !== undefined) {
                lineDiv.innerText = `PENDING: ${result.skipped} - ${result.object}; size ${Math.round(((result.size / (1024 * 1024)) + Number.EPSILON) * 100) / 100}MB`;
                hasSeenOperations = true;
            } else if (!hasSeenOperations) {
                lineDiv.innerText = result.msg;
            } else {
                continue;
            }
        } else {
            lineDiv.innerText = result.msg;
        }

        lineContEl.appendChild(lineDiv);
    }

    if (!dryRun) {
        const lineDiv = document.createElement("div");
        lineDiv.className = "bisync-line";
        lineDiv.innerText = "Sync Complete!";
        lineContEl.appendChild(lineDiv);    
    }

    const closeSyncButtonEl = document.getElementById('close-sync-window');
    closeSyncButtonEl.style.display = 'block';
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

    const closeSyncButtonEl = document.getElementById('close-sync-window');
    closeSyncButtonEl.style.display = 'block';

    syncConfirm.innerText = "Retry";
    retryDryRun = dryRun;

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
    pendingSyncGame = null;
    const closeSyncButtonEl = document.getElementById('close-sync-window');
    closeSyncButtonEl.style.display = 'block';
    retryDryRun = false;
    syncConfirm.innerText = "Confirm";
}

async function sync(gameName, dryRun) {
    const closeSyncButtonEl = document.getElementById('close-sync-window');
    closeSyncButtonEl.style.display = 'none';
    log(`Checking If should perform dry run - ${dryRun}`);
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    const syncConfirm = document.getElementById('sync-modal-confirm');
    const syncCancel = document.getElementById('sync-modal-cancel');
    syncConfirm.style.display = 'none';
    syncCancel.style.display = 'none';

    if (dryRun) {
        bisyncSubtitle.innerText = "Simulating transfer - please wait";
        await getSyncDryRun(gameName);
    } else {
        bisyncSubtitle.innerText = "Performing sync - please wait";
        await syncGame(gameName);
    }

    const timerValue = 250;
    var timerFunc = async () => {
        var clear = false;
        let resultStr = "";
        try {
            resultStr = await pollLogs(gameName);
        } catch (error) {
            await onSyncError(error, dryRun);
            return;
        }

        if (resultStr !== "") {
            const result = JSON.parse(resultStr);
            if (result && result.Finished) {
                clear = true;
                try {
                    await onFinished(result, dryRun);
                } catch (e) {
                    await log(`Error : ${e}`);
                } finally {
                    if (dryRun) {
                        syncConfirm.style.display = 'block';
                        syncCancel.style.display = 'block';    
                    }
                }
            }    
        }

        if (!clear) {
            setTimeout(timerFunc, timerValue);
        }
    };

    setTimeout(timerFunc, timerValue);
}

async function setupSyncModal(gameName) {
    log(`Setting up sync for ${gameName}`);
    resetSyncModal();
    document.getElementById('accordion-cont').style.display='none';

    pendingSyncGame = gameName;
    const shouldPerformDryRun = await getShouldPerformDryRun();
    await sync(gameName, shouldPerformDryRun);
}

async function onSyncGameModalClosed(element) {
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
    const dryRun = retryDryRun;
    retryDryRun = false;
    await sync(pendingSyncGame, dryRun);
}