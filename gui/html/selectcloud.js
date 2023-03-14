const CloudSelectionState = {
    pendingCloudInitialization: false,
};


async function closeSelectCloud(element) {
    CloudSelectionState.pendingCloudInitialization = false;
    refresh();
}

// @TODO if you close the window, this will not kill the pending rclone
// process, and subsequent syncs will fail. 
async function cloudSelected(cloudService) {
    if (CloudSelectionState.pendingCloudInitialization) {
        await cancelPendingCloudSelection();
    }

    const closeModal = document.getElementById("closemodal");
    closeModal.style.display = 'none';

    await setCloud(cloudService, " (Pending)");
    CloudSelectionState.pendingCloudInitialization = true;
    await commitCloudService(cloudService);

    const timer = 250;
    const poll = async () => {
        const result = await isCloudSelectionComplete()
                        .catch(e => {
                            log(`Error setting up cloud ${e}`);
                        });
        if (result) {
            refresh();
        } else {
            setTimeout(poll, timer);

        }
    };
    setTimeout(poll, timer);
}


async function setCurrentCloud() {
    const service = await getCloudService();
    await setCloud(service);
}

async function setCloud(service, postfix = "") {
    const currentCloudEl = document.getElementById("currentcloudcont");
    const closeModal = document.getElementById("closemodal");
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
        currentCloudEl.innerText = prefix + value + postfix;
    }
}

function setNextCloud() {
    const nextCloudModal = document.getElementById('nextcloud-modal');
    nextCloudModal.style = 'display: block';
}

async function onNextCloudConfirm() {
    const url = document.getElementById('NextCloud-Url');
    const userName = document.getElementById('NextCloud-Username');
    const password = document.getElementById('NextCloud-Password');
    const bearerToken = document.getElementById('NextCloud-BearerToken');
    const nextCloudSettings = {
        url: url.value,
        user: userName.value,
        bearer_token: bearerToken.value,
        pass: password.value,
    };

    await deleteCurrentNextCloudSettings();
    await commitNextCloudSettings(JSON.stringify(nextCloudSettings));
    await cloudSelected(4);
}

async function onNextCloudClose() {
    const nextCloudModal = document.getElementById('nextcloud-modal');
    nextCloudModal.style = 'display: none';

    const url = document.getElementById('NextCloud-Url');
    const userName = document.getElementById('NextCloud-Username');
    const password = document.getElementById('NextCloud-Password');
    const bearerToken = document.getElementById('NextCloud-BearerToken');
    url.value = "";
    userName.value = "";
    password.value = "";
    bearerToken.value = "";
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

    await deleteCurrentFTPSettings();
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