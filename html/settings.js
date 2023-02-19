async function onSettingsModalClosed(element) {
    document.getElementById('accordion-cont').style.display='block';
    const settingsEl = document.getElementById('settings-modal');
    settingsEl.style.display = 'none';
}

async function onSettingsModalOpen() {
    document.getElementById('accordion-cont').style.display='none';
    const currentSettingsString = await getCloudPerfs();
    const currentSettings = JSON.parse(currentSettingsString);

    const settingsEl = document.getElementById('settings-modal');
    settingsEl.style.display = 'block';

    const dryRunSwitch = document.getElementById('settings-dry-run');
    dryRunSwitch.checked = currentSettings.performDryRun;

    const syncSwitch = document.getElementById('settings-use-sync');
    syncSwitch.checked = currentSettings.useBiSync;
}

async function onDryRunToggle(element) {
    const dryRunSwitch = document.getElementById('settings-dry-run');
    const currentSettingsString = await getCloudPerfs();
    const currentSettings = JSON.parse(currentSettingsString);

    currentSettings.performDryRun = dryRunSwitch.checked;
    await commitCloudPerfs(JSON.stringify(currentSettings));
}

async function onUseBiSyncToggle(element) {
    const syncSwitch = document.getElementById('settings-use-sync');
    const currentSettingsString = await getCloudPerfs();
    const currentSettings = JSON.parse(currentSettingsString);

    currentSettings.useBiSync = syncSwitch.checked;
    await commitCloudPerfs(JSON.stringify(currentSettings));
}