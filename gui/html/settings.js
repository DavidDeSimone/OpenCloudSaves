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

    const doNotPromptSwitch = document.getElementById('settings-should-not-prompt-large');
    doNotPromptSwitch.checked = currentSettings.shouldNotPromptForLargeSyncs;
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

async function onDoNotPromptLargeSyncsToggle() {
    const syncSwitch = document.getElementById('settings-should-not-prompt-large');
    const currentSettingsString = await getCloudPerfs();
    const currentSettings = JSON.parse(currentSettingsString);

    currentSettings.shouldNotPromptForLargeSyncs = syncSwitch.checked;
    await commitCloudPerfs(JSON.stringify(currentSettings));
}

async function onNoticeClicked() {
    const noticeModal = document.getElementById('notice-modal');
    noticeModal.style.display = 'block';
}

async function onNoticeClosed() {
    const noticeModal = document.getElementById('notice-modal');
    noticeModal.style.display = 'none';
}

async function onDeleteAllDrivesClicked() {
    window.OnDeleteAllDrivesError = () => {

    };

    window.OnDeleteAllDrivesComplete = () => {
        initializeGui();
    };

    makeConfirmationPopup({
        title: "Reset RClone config",
        subtitle: "Are you sure you want to reset your rclone config for OpenCloudSave? This will only reset the cloud config's that OpenCloudSave defined. Your save data will not be affected",
        onConfirm: async () => {
            await deleteAllDrives();
        }
    })
}

