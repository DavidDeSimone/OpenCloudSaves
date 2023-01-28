async function cloudSelected(cloudService) {
    await log("Selected " + cloudService);
    await commitCloudService(cloudService);
    refresh();
}

async function setCurrentCloud() {
    const currentCloudEl = document.getElementById("currentcloudcont");
    const service = await getCloudService();
    const prefix = "Current Cloud Storage: ";
    let value = "";

    switch (service) {
        case -1: 
            currentCloudEl.style.display = 'none';
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
        default:
            currentCloudEl.style.display = 'none';
            break;
    }

    if (value !== "") {
        currentCloudEl.innerText = prefix + value;
    }
} 

setCurrentCloud();