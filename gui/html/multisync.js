const MultiSyncState = {
    hasActiveSyncOperation: false,
    dryRunComplete: false,
    gameToSync: null,
    pendingCancel: false,
};

async function onOpenMultisync(element) {
    const modal = document.getElementById('multisync-modal');
    const subTitle = document.getElementById('multisync-subtitle');
    const multisync = document.getElementById('multisync-line-cont');
    modal.style.display = 'block';
    subTitle.innerText = "Please select and view what games you would like to sync.";
    multisync.innerHTML = "";
    MultiSyncState.dryRunComplete = false;
    MultiSyncState.hasActiveSyncOperation = false;
    MultiSyncState.pendingCancel = false;
    MultiSyncState.gameToSync = null;

    const gameNamesJson = await getMultisyncSelectedGames();
    const gameNames = JSON.parse(gameNamesJson);

    const checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        const name = check.id.replaceAll("-multisync-check", "");
        const spinner = document.getElementById(`${name}-multisync-game-modal-loader`);
        const success = document.getElementById(`${name}-multisync-success`);
        const failure = document.getElementById(`${name}-multisync-failure`);

        spinner.style.display = 'none';
        success.style.display = 'none';
        failure.style.display = 'none';
        check.checked = gameNames[name] !== undefined;
    }
}

async function resetSuccessAndFailureMarkers() {
    const checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        const name = check.id.replaceAll("-multisync-check", "");
        const success = document.getElementById(`${name}-multisync-success`);
        const failure = document.getElementById(`${name}-multisync-failure`);

        success.style.display = 'none';
        failure.style.display = 'none';
    }
}

async function multisyncConfirmationCancellation() {
    makeConfirmationPopup({
        title: `Are you sure you want to cancel this operation?`,
        subtitle: `If this was not a dry-run, your sync may be partially complete.`,
        onConfirm: async () => {
            if (MultiSyncState.gameToSync !== '') {
                await cancelPendingSync(MultiSyncState.gameToSync);
            }
            
            MultiSyncState.gameToSync = null;
            MultiSyncState.hasActiveSyncOperation = false;
            MultiSyncState.pendingCancel = true;
            refresh();
        },
    });
}

async function onCloseMultisync(element) {
    if (MultiSyncState.hasActiveSyncOperation) {
        await multisyncConfirmationCancellation();
        return;
    }


    const modal = document.getElementById('multisync-modal');
    modal.style.display = 'none';
    MultiSyncState.dryRunComplete = false;
    MultiSyncState.hasActiveSyncOperation = false;

    const gameNames = [];
    const checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        const name = check.id.replaceAll("-multisync-check", "");
        if (check.checked) {
            gameNames.push(name);
        }
    }

    commitMultisyncSelectGames(gameNames);
}

async function onMultisyncSelectAllClicked() {
    var checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        check.checked = true;
    }
}

async function onMultisyncUnselectAllClicked() {
    var checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        check.checked = false;
    }
}

async function onSyncSelectedClicked() {
    const multisyncInput = document.getElementById('multisync-input');
    multisyncInput.value = "";
    await onChangeSearchMultisync(multisyncInput);
    await resetSuccessAndFailureMarkers();


    const dryRunSettings = await getShouldPerformDryRun();
    const dryRun = dryRunSettings && !MultiSyncState.dryRunComplete;

    const multisync = document.getElementById('multisync-line-cont');
    multisync.innerHTML = "";

    const gamesToSync = [];
    var checks = document.getElementsByClassName("multisync-check");
    for (let i = 0; i < checks.length; ++i) {
        const check = checks[i];
        if (check.checked) {
            var name = check.id.replaceAll("-multisync-check", "");
            gamesToSync.push(name);

            var spinner = document.getElementById(`${name}-multisync-game-modal-loader`);
            spinner.style.display = 'block';
            check.disabled = true;
            check.style.display = 'none';
        }
    }

    const checkSpans = document.getElementsByClassName('multisync-checkmark');
    for (let i = 0; i < checkSpans.length; ++i) {
        checkSpans[i].style.display = 'none';
    }
 
    const multisyncButton = document.getElementById('multisync-modal-confirm');
    multisyncButton.disabled = true;
    MultiSyncState.hasActiveSyncOperation = true;

    for (let i = 0; i < gamesToSync.length; ++i) {
        if (MultiSyncState.pendingCancel) {
            return;
        }

        const gameName = gamesToSync[i]; 
        try {
            MultiSyncState.gameToSync = gameName;
            await performSingleGameSync(gameName, dryRun);
        } catch(e) {
            await onSyncGameFailure(gameName);
            continue;
        } finally {
            MultiSyncState.gameToSync = null;
        }

        await onSyncGameComplete(gameName);
    }

    MultiSyncState.hasActiveSyncOperation = false;

    for (let i = 0; i < checkSpans.length; ++i) {
        checkSpans[i].style.display = 'block';
    }

    const subTitle = document.getElementById('multisync-subtitle');
    subTitle.innerText = dryRun ? `Your Data is not sync'd. Please review your dry run results and press the sync button if you would like to sync to the cloud.` : `Sync Complete!`;

    // If we have just completed our sync post-dryrun,
    // we will reset dryRunComplete.
    if (MultiSyncState.dryRunComplete) {
        MultiSyncState.dryRunComplete = false;
    }

    // Since this was a dry-run, we will mark dryRunComplete
    // as true so that we will actually sync next run.
    if (dryRun) {
        MultiSyncState.dryRunComplete = true;
    }

    multisyncButton.disabled = false;
}

async function onSyncGameComplete(gameName) {
    const success = document.getElementById(`${gameName}-multisync-success`);
    const spinner = document.getElementById(`${gameName}-multisync-game-modal-loader`);
    success.style.display = 'block';
    spinner.style.display = 'none';
}

async function onSyncGameFailure(gameName) {
    const success = document.getElementById(`${gameName}-multisync-failure`);
    const spinner = document.getElementById(`${gameName}-multisync-game-modal-loader`);
    success.style.display = 'block';
    spinner.style.display = 'none';

}

function recordMessage(message) {
    const multisync = document.getElementById('multisync-line-cont');
    const lineDiv = document.createElement('div');
    lineDiv.className = "bisync-line";
    lineDiv.innerText = message;
    multisync.appendChild(lineDiv);
}

async function processPoll(gameName) {
    const logsStr = await pollLogs(gameName);
    if (logsStr == "") {
        return false;
    }
    
    const result = JSON.parse(logsStr);
    if (result && result.Finished) {
        recordMessage(result.Message)
        return true;
    }

    const multisync = document.getElementById('multisync-line-cont');
    multisync.style.display = 'block';                
    const messages = (result && result.Message) ? result.Message.split("\n") : [];
    for (let i = 0; i < messages.length; ++i) {
        const message = messages[i];
        if (message === null || message === "") {
            continue;
        }

        recordMessage(message);
    }

    return false;
}

async function pollLoop(gameName) {
    return new Promise((resolve, reject) => {
        const logTime = 250;
        const func = async () => {
            let errorState = false;
            const complete = await processPoll(gameName)
                                    .catch((error) => {
                                        errorState = true;
                                        recordMessage(error);
                                        reject();
                                    });
            if (errorState) {
                return;
            } else if (complete) {
                resolve();
            } else {
                setTimeout(func, logTime);
            }
        };
        const timeout = setTimeout(func, logTime);
    });
}

async function performSingleGameSync(gameName, dryRun) {
    const subTitle = document.getElementById('multisync-subtitle');
    subTitle.innerText = `Performing sync for ${gameName}`;

    const prefix = dryRun ? "Performing Dry Run" : "Syncing";
    const suffix = dryRun ? "Dry Run Complete" : "Sync Complete";

    const multisync = document.getElementById('multisync-line-cont');
    multisync.style.display = 'block';

    recordMessage(`---------------------------------------------------------`);
    recordMessage(`${prefix}: ${gameName}`);
    recordMessage(`---------------------------------------------------------`);
    if (dryRun) {
        await getSyncDryRun(gameName);
    } else {
        await syncGame(gameName);
    }

    await pollLoop(gameName)
    .finally(() => {
        recordMessage(`---------------------------------------------------------`);
        recordMessage(`${suffix}: ${gameName}`);
        recordMessage(`---------------------------------------------------------`);
    
    });
}

async function sleepFor(miliseconds) {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve();
        }, miliseconds)
    });
}

async function onChangeSearchMultisync(element) {
    const name = element.value

    var acc = document.getElementsByClassName("multisync-container");
    var i;
    
    for (i = 0; i < acc.length; i++) {
        if (name === "") {
            acc[i].style.display = "block";
            continue;
        }

        const res = fuzzyMatch(name.toLowerCase(), acc[i].id.replace("-cont", "").toLowerCase())
        if (res[0] || res[1] > DEFAULT_SEARCH_SCORE) {
            acc[i].style.display = "block";
        } else {
            acc[i].style.display = "none";
        }
    }
}