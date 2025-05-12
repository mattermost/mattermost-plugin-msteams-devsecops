# Mattermost Mission Collaboration for Microsoft 365

[![Build Status](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/ci.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/actions/workflows/ci.yml)

**Mattermost Mission Collaboration for Microsoft** is a plugin that embeds Mattermost directly inside Microsoft 365, Teams, and Outlook clients. This integration extends Microsoft 365, Teams, and Outlook for mission-critical coordination, command and control, incident response, and DevSecOps workflows in demanding environments, including air-gapped and classified networks. 

> [!NOTE]  
> This product is currently in **Beta**. We're excited to share it with you and welcome your feedback to help us improve. While the core features are ready for use, you may encounter minor issues as we continue to refine the experience. Please share your thoughts and suggestions in the [~user-feedback](https://community.mattermost.com/core/channels/user-feedback) channel or submit an issue on [GitHub](https://github.com/mattermost/mattermost-plugin-msteams-devsecops/issues).

## Mattermost Mission Collaboration for Microsoft 365 and MS Teams

**Benefits & Use Cases:**
- **External Collaboration with IT Control**: Replace non-compliant freemium personal apps with dedicated external collaboration across mobile, web, and desktop, fully controlled by IT.
- **Intelligent, AI-Accelerated Incident Response**: Augment Microsoft Security Suite with AI-powered collaborative workflows, from detection to resolution, within secure environments. 
- **Sovereign, Cyber-Resilient S4B Replacement for Classified Workflows**: Replace legacy Skype for Business with a self-hosted, fully sovereign solution for classified operations, tightly integrated within Microsoft ecosystems. 
- **Embedded DevSecOps Collaboration Inside Microsoft Teams**: Maintain a unified user experience while achieving higher operational productivity for DevSecOps and mission teams.
- **Mission Operations at the Tactical Edge**: Real-time command and control for joint operations, mission partner environments, and disconnected/denied environments (DDIL). 

**Features:**
- **Direct Access**: Access Mattermost directly from a tab without switching applications or opening a browser. 
- **Seamless Integration**: Experience the full functionality of Mattermost within a familiar Microsoft Teams interface. 
- **Real-time Collaboration**: Collaborate with your team on projects, workflows, and communications in real time. 
- **Unified Communications**: Combine chat, meetings, workflows, and task management inside MS Teams. 
- **Secure Data Handling**: Maintain data sovereignty with self-hosted deployment options. 
- **AI-Powered Insights**: Use multi-agent AI, including Azure OpenAI and local LLMs, for faster decision-making and situational awareness. 
- **Embedded DevSecOps Collaboration**: Keep developer teams productive with integrated workflows inside Microsoft Teams.
- **Entra-Based SSO**: Simplify user authentication and enhance security with enterprise-grade identity management for organizations using Microsoft Entra ID.
- **Activity Feed Notifications for Mentions in Mattermost**: Never miss critical updates, with real-time notifications in your MS Teams activity feed whenever someone mentions you in Mattermost.

This app is designed to work with Microsoft 365, Outlook, and Microsoft Teams.

### Admin Setup

For detailed setup instructions, see the [Setup Guide](https://docs.mattermost.com/integrate/mattermost-mission-collaboration-for-m365.html#setup), which provides step-by-step instructions for creating and configuring a Microsoft Teams application in Azure, setting up the Mattermost plugin, and installing Mattermost in Microsoft Teams.

### FAQ

#### Where can I get support?

You can browse existing open issues or submit a new issue for support in GitHub: https://github.com/mattermost/mattermost-teams-tab/issues

#### How do I fix a 404 error in Teams/Outlook tab?

If after following the [Setup Guide](https://docs.mattermost.com/integrate/mattermost-mission-collaboration-for-m365.html#setup), you see a 404 error in the MS Teams tab, it likely means the plugin did not start correctly on your Mattermost instance. Check the plugin configuration for any missing or incorrect settings. Also change the [Site URL](https://docs.mattermost.com/configure/environment-configuration-settings.html#site-url) is correct and reachable for your Mattermost instance.

#### What do I do if my users cannot see the app I deployed?

If after following the [Setup Guide](https://docs.mattermost.com/integrate/mattermost-mission-collaboration-for-m365.html#setup), your users cannot see the M365 app uploaded to your app store, you may simply need to wait. Microsoft states this can take up to 24 hours, however our experience has been the delay is anywhere from seconds to a few hours. If after 24 hours users still cannot see the app, you may need to remove it and upload it again.
