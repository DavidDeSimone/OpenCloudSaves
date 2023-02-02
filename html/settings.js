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
}

async function onSettingsToggle(element) {
    const dryRunSwitch = document.getElementById('settings-dry-run');
    const currentSettingsString = await getCloudPerfs();
    const currentSettings = JSON.parse(currentSettingsString);

    currentSettings.performDryRun = dryRunSwitch.checked;
    await commitCloudPerfs(JSON.stringify(currentSettings));
}