

<p align="center">
<img width="550" alt="Screenshot 2023-02-04 at 6 01 24 PM" src="https://user-images.githubusercontent.com/7245174/216797736-dbf39f33-2ce4-4ef5-9565-deb24aa95952.png">
</p>



Open Cloud Saves is an open source application for managing your saves games across Windows, MacOs, and Linux (including SteamOS). Open Cloud Saves is available for use offically as a "beta". As a beta test, we recommend that you manually make a backup of your save data before usage. Until Open Cloud Save is more battle tested, we will issue a warning for users to use caution with "critical, beloved" save data. 

Open Cloud Save gives an advantage over exisiting cloud solutions:

* Allows cloud saves for games without developer support
* Allows for the exclusion of certain files or filetypes. This can prevent games syncing graphical settings in addition to syncing save data. 
* Allows for sync between storefronts (you own a Steam on linux and a Epic Game Store version on windows

We have a growing list of game save locations, which we will upload to and from your exisiting cloud provider:


<p align="center">
  <img width="798" alt="Screenshot 2023-02-04 at 6 06 16 PM" src="https://user-images.githubusercontent.com/7245174/216797838-82411ebf-3bba-46bb-a277-9d6e35595806.png">
 </p>

Under the hood, OpenCloudSave uses the popular tool [rclone](https://github.com/rclone/rclone) to perform a bi-directional sync to allow your for games to be updated across multiple clients. This includes Valve's Steam Deck. 



In addition to our provided library of game save locations, you can define custom save locations for your games. If you have a game you would like added to the default list, be sure to submit an issue https://github.com/DavidDeSimone/OpenCloudSaves/issues


