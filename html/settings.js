async function onSettingsModalClosed(element) {
    await log("Closing.....");
    const settingsEl = document.getElementById('settings-modal');
    settingsEl.style.display = 'none';
}

async function onSettingsModalOpen() {
    const settingsEl = document.getElementById('settings-modal');
    settingsEl.style.display = 'block';

}