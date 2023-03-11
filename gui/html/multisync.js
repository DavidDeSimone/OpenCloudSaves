async function onOpenMultisync(element) {
    const modal = document.getElementById('multisync-modal');
    modal.style.display = 'block';
}

async function onCloseMultisync(element) {
    const modal = document.getElementById('multisync-modal');
    modal.style.display = 'none';
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
    const dryRunSettings = await getShouldPerformDryRun();

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

    for (let i = 0; i < gamesToSync.length; ++i) {
        const gameName = gamesToSync[i]; 
        try {
            await performSingleGameSync(gameName, dryRunSettings);
        } catch(e) {
            await onSyncGameFailure(gameName);
            continue;
        }
        await onSyncGameComplete(gameName);
    }

    for (let i = 0; i < checkSpans.length; ++i) {
        checkSpans[i].style.display = 'block';
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
    // multisync.style.display = 'block';
    const lineDiv = document.createElement('div');
    lineDiv.className = "bisync-line";
    lineDiv.innerText = message;
    multisync.appendChild(lineDiv);
}

async function pollLoop(gameName) {
    return new Promise((resolve, reject) => {
        const logTime = 250;
        const func = async () => {
            try {
                let logsStr = "";
                try {
                    logsStr = await pollLogs(gameName);
                } catch (e) {
                    recordMessage(e);
                    throw e;
                }

                if (logsStr == "") {
                    setTimeout(func, logTime);
                    return;
                }
                
                const result = JSON.parse(logsStr);
                if (result && result.Finished) {
                    recordMessage(result.Message)
                    resolve();
                    return;
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

                setTimeout(func, logTime);
            } catch (e) {
                await log(`Top level Error: ${e}`);
                reject(e);
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

    recordMessage(`---------------------------------------------------------`);
    recordMessage(`${prefix}: ${gameName}`);
    recordMessage(`---------------------------------------------------------`);
    if (dryRun) {
        await getSyncDryRun(gameName);
    } else {
        await syncGame(gameName);
    }
    await pollLoop(gameName);
    recordMessage(`---------------------------------------------------------`);
    recordMessage(`${suffix}: ${gameName}`);
    recordMessage(`---------------------------------------------------------`);
}

async function sleepFor(miliseconds) {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve();
        }, miliseconds)
    });
}