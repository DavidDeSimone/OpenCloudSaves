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
    if (dryRunSettings) {
        // @TODO warn user this screen doesn't perform dry runs
    }

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
            await performSingleGameSync(gameName);
        } catch(e) {
            await onSyncGameFailure(gameName);
            continue;
        }
        await onSyncGameComplete(gameName);
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

async function pollLoop(gameName) {
    return new Promise((resolve, reject) => {
        const func = async () => {
            try {
                const logsStr = await pollLogs(gameName);
                const result = JSON.parse(logsStr);
                if (result && result.Finished) {
                    resolve();
                }
                setTimeout(func, 500);
            } catch (e) {
                reject(e);
            }
        };
        const timeout = setTimeout(func, 500);
    });
}

async function performSingleGameSync(gameName, dryRun) {
    const subTitle = document.getElementById('multisync-subtitle');
    subTitle.innerText = `Performing sync for ${gameName}`;
    await log(`Syncing ${gameName}`);
    await syncGame(gameName);
    await pollLoop(gameName);
    await log(`Sync for ${gameName} complete`);
}

async function sleepFor(miliseconds) {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve();
        }, miliseconds)
    });
}