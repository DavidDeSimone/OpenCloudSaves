// window.pendingSyncGame = null;
let pendingSyncGame = null;

async function onFinished(result, dryRun) {
    log("Finished Sync");
    const messages = result.Message.split("\n");
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

        const result = JSON.parse(messages[i]);
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
}

async function sync(gameName, dryRun) {
    log(`Checking If should perform dry run - ${dryRun}`);
    const bisyncSubtitle = document.getElementById('bisync-subtitle');
    if (dryRun) {
        bisyncSubtitle.innerText = "Simulating transfer - please wait";
        await getSyncDryRun(gameName);
    } else {
        bisyncSubtitle.innerText = "Performing sync - please wait";

        const syncConfirm = document.getElementById('sync-modal-confirm');
        const syncCancel = document.getElementById('sync-modal-cancel');
        syncConfirm.style.display = 'none';
        syncCancel.style.display = 'none';
        await syncGame(gameName);
    }

    const interval = setInterval(async () => {
        const resultStr = await pollLogs(gameName);
        if (resultStr !== "") {
            const result = JSON.parse(resultStr)
            if (result && result.Finished) {
                clearInterval(interval);
                try {
                    await onFinished(result, dryRun);
                } catch (e) {
                    await log(`Error : ${e}`);
                }
            }    
        }
    }, 500);
}

async function setupSyncModal(gameName) {
    log(`Setting up sync for ${gameName}`);
    resetSyncModal();

    pendingSyncGame = gameName;
    const shouldPerformDryRun = await getShouldPerformDryRun();
    await sync(gameName, shouldPerformDryRun);
}

async function onSyncGameModalClosed(element) {
    const model = document.getElementById("bisync-confirm");
    model.style.display = 'none';
}

async function onSyncGameConfirm(element) {
    const loaderEl = document.getElementById('sync-game-modal-loader');
    const lineContEl = document.getElementById('bisync-line-cont');

    loaderEl.style.display = 'block';
    lineContEl.innerHTML = "";
    await sync(pendingSyncGame, false);
}