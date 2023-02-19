async function closeSelectCloud(element) {
    refresh();
}

async function cloudSelected(cloudService) {
    await log("Selected " + cloudService);
    await commitCloudService(cloudService);
    refresh();
}

async function setCurrentCloud() {
    const currentCloudEl = document.getElementById("currentcloudcont");
    const closeModal = document.getElementById("closemodal");
    const service = await getCloudService();
    const prefix = "Current Cloud Storage: ";
    let value = "";

    switch (service) {
        case -1: 
            currentCloudEl.style.display = 'none';
            closeModal.style.display = 'none';
            break;
        case 0:
            value = "Google Cloud";
            break;
        case 1:
            value = "One Drive";
            break;
        case 2:
            value = "Drop Box";
            break;
        case 3:
            value = "Box";
            break;
        case 4: 
            value = "Next Cloud";
            break;
        case 5:
            value = "Custom FTP Server";
            break;
        default:
            currentCloudEl.style.display = 'none';
            closeModal.style.display = 'none';
            break;
    }

    if (value !== "") {
        currentCloudEl.innerText = prefix + value;
    }
}

function setFtpServer() {
    const ftpModal = document.getElementById('ftp-modal');
    ftpModal.style = 'display: block';
}

async function onFtpConfirm() {
    const hostName = document.getElementById('HostName');
    const port = document.getElementById('Port');
    const userName = document.getElementById('UserName');
    const password = document.getElementById('Password');
    const ftpSettings = {
        host: hostName.value,
        port: port.value,
        userName: userName.value,
        password: password.value,
    };

    await commitFTPSettings(JSON.stringify(ftpSettings));
    await cloudSelected(5);
}

async function onFtpClose() {
    const ftpModal = document.getElementById('ftp-modal');
    ftpModal.style = 'display: none';

    const hostName = document.getElementById('HostName');
    const port = document.getElementById('Port');
    const userName = document.getElementById('UserName');
    const password = document.getElementById('Password');
    hostName.value = "";
    port.value = "";
    userName.value = "";
    password.value = "";
}

setCurrentCloud();