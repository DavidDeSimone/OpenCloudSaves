async function cloudSelected(cloudService) {
    await log("Selected " + cloudService);
    await commitCloudService(cloudService);
    refresh();
}