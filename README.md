# Mattermost DevSecOps for Microsoft Teams

[![Build Status](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/ci.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/ci.yml)
[![E2E Status](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/e2e.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/e2e.yml)

This repository provides the foundation for multiple Microsoft App offerings that integrate with the Mattermost platform.

- **Community Mattermost for Microsoft 365 & MS Teams:** Free offering to explore Mattermost capabilities and meet with fellow end users, customers and evaluators, along with Mattermost staff.
- **DevSecOps Mattermost for Microsoft 365 & MS Teams:** Offering available to active Mattermost customers as an added benefit to their subscription for accessing Mattermost from their Microsoft Teams, Outlook at M365 user experiences.

## Community Mattermost for Microsoft 365 & MS Teams

The *Community Mattermost for Microsoft 365* application provides a showcase and peer-to-peer discussion environment for Mattermost customers, open source users, and system evaluators.

Community Mattermost runs as an online service at https://community.mattermost.com and made available in the Microsoft Teams, Microsoft Outlook and Microsoft Application hosting environment with this offering.

Capabilities:
- Connect to Mattermost Community environment as a Microsoft Application from within Microsoft Teams and Outlook web and desktop environments.

Benefits:
- Seamlessly communicate with Mattermost Community from Microsoft Teams and Outlook with use of application tabs.
- Evaluate a showcase deployment of Mattemost capabilities in consideration of self-hosting the platform within your Azure or on-prem environments.  
- Share input with Mattermost staff and developers on future improvements to the platform.

The following future capabilities are being considered for addition in upcoming releases:

- Integrated notifications across Mattermost and Microsoft Teams.
- Optional Single-Sign-On with integrated authentication.

**Key Features:**  
- **Direct Access**: Access the Mattermost Customer Community directly from a tab without switching applications or opening a browser  
- **Seamless Integration**: Experience the full functionality of the Mattermost Customer Community within a familiar interface  
- **Real-time Collaboration**: Engage with the Mattermost community in real-time discussions on product features, technical questions, and best practices  
- **Professional Support**: Licensed customers can get help from Mattermost staff
- **Peer-to-Peer Q&A**: All customers and users of Mattermost free and open source editions can get peer-to-peer help and input at no cost
- **Contribute to Development**: Participate in discussions that shape the future direction of Mattermost products  
- **Knowledge Sharing**: Learn implementation strategies and best practices from a diverse community of users  
- **Stay Updated**: Keep up with the latest Mattermost announcements, updates, and roadmap information  

This app is designed to work with Microsoft 365, Outlook, and Microsoft Teams. A free account is required to use the Mattermost Customer Community.  

**About Mattermost:**  
Mattermost is a purpose-built platform for technical and operational teams working in organizations vital to national security, public safety, and critical infrastructure. [https://mattermost.com/](https://mattermost.com/)  

## DevSecOps Mattermost for Microsoft 365 & MS Teams

This project is under development for customers of Mattermost Professional and Enterprise subscriptions for advanced capabilities within Microsoft 365 and MS Teams.  

## Create and set up a Teams application in Azure

1. Go to your **Azure Portal > Microsoft Entra ID**

2. Go to **App registrations**

3. Create a new registration
    - Give it a name
    - Accounts in this organisational directory only (single tenant)
    - No redirect URIs

4. Go to your newly created application
    - Make note of these values as we’ll need those later:
        - Application (client) ID → _Required in the plugin configuration_
        - Directory (tenant) ID → _Required for the plugin configuration_
        - Object ID → _Required in the plugin configuration_

5. Go to **Certificates and secrets**
    - Generate a new Client secret
    - Make note of the secret value. → _Required in the plugin configuration_

6. Go to **API Permissions**
    - Ensure `User.Read` **delegated** permission is added ([Microsoft documentation](https://learn.microsoft.com/en-us/microsoftteams/platform/tabs/how-to/authentication/tab-sso-register-aad#enable-sso-in-microsoft-entra-id))
    - Add `TeamsActivity.Send` **application** permission (optional, for notifications) ([Microsoft documentation](https://learn.microsoft.com/en-us/graph/teams-send-activityfeednotifications?tabs=desktop%2Chttp))
    - Grant admin consent for the default directory to prevent users from getting the consent prompt.

7. Go to **Expose an API**
    - Edit the “_Application ID URI_” as such: `api://{{Mattermost Site URL Hostname}}/{{Application (client) ID}}`
    - Add the `access_as_user` scope by clicking the “Add a scope” button. ([Microsoft documentation](https://learn.microsoft.com/en-us/microsoftteams/platform/tabs/how-to/authentication/tab-sso-register-aad#to-configure-api-scope))
        - **Scope name**: `access_as_user`
        - **Who can consent?** Admins and users
        - Give it a display name and description, and also specify a user consent display name and description. These last two are the ones the end users are going to see in the consent screen. For example:
            **Display name**: Log in to Mattermost
            **Description**: Used to allow O365 users to log in to the Mattermost application 
            **User consent display name**: Log in to Mattermost
            **User consent description**: This permission is required to automatically log you in into Mattermost from Microsoft applications.
        - Add authorised client applications for the scope we just created ([Microsoft documentation](https://learn.microsoft.com/en-us/microsoftteams/platform/tabs/how-to/authentication/tab-sso-register-aad#to-configure-authorized-client-application))
            - Click on “_Add a client application_”
                - **Authorised scopes**: The one we just created
                - **Client ID**:
                    - **Teams web**: 5e3ce6c0-2b1f-4285-8d4b-75ee78787346
                    - **Teams app**: 1fec8e78-bce4-4aaf-ab1b-5451cc387264
                - (You need to add a client application per target Microsoft application you want)
                - If you want to make your application available in more Microsoft application, you need to keep adding client applications from [the following table](https://learn.microsoft.com/en-us/microsoftteams/platform/tabs/how-to/authentication/tab-sso-register-aad#to-configure-authorized-client-application:~:text=Select%20one%20of%20the%20following%20client%20IDs%3A).
8. Go to your **Mattermost server's system console > Plugins > MSTeams DevSecOps**:
    - Enter the values you made note earlier in the appropriate fields:
        - **Application (client) ID**: The Application (client) ID you noted from step 4
        - **Directory (tenant) ID**: The Directory (tenant) ID you noted from step 4
        - **Object ID**: The Object ID you noted from step 4
        - **Client Secret**: The secret value you generated in step 5
    - Save the changes and enable the plugin.