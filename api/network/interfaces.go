package network

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// NetworkAPIClient defines the interface for UniFi Network API operations.
// This interface enables consumers to create mock implementations for testing.
//
// The Network API provides access to a local UniFi controller for managing:
//   - Sites and devices
//   - Network clients
//   - DNS records
//   - Firewall policies
//   - Traffic rules (QoS)
//   - Hotspot vouchers
//   - Dashboard statistics
//
// All methods mirror the corresponding methods in APIClient to ensure
// compatibility and ease of use.
//
// Example usage with mocking frameworks:
//
//	// Using gomock:
//	//go:generate mockgen -destination=mocks/network_client.go -package=mocks github.com/lexfrei/go-unifi/api/network NetworkAPIClient
//
//	// Using testify/mock:
//	type MockClient struct {
//	    mock.Mock
//	}
//
//	func (m *MockClient) ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error) {
//	    args := m.Called(ctx, site)
//	    return args.Get(0).([]DNSRecord), args.Error(1)
//	}
//
//nolint:revive // NetworkAPIClient is intentionally explicit to avoid confusion with APIClient struct
type NetworkAPIClient interface { //nolint:interfacebloat // This interface mirrors the full API client with 22 methods
	// Sites operations

	// ListSites retrieves a list of all sites configured on the controller.
	ListSites(ctx context.Context, params *ListSitesParams) (*SitesResponse, error)

	// Devices operations

	// ListSiteDevices retrieves a list of all devices for a specific site.
	ListSiteDevices(ctx context.Context, siteID SiteId, params *ListSiteDevicesParams) (*DevicesResponse, error)

	// GetDeviceByID retrieves detailed information about a specific device.
	GetDeviceByID(ctx context.Context, siteID SiteId, deviceID DeviceId) (*Device, error)

	// Clients operations

	// ListSiteClients retrieves a list of all clients for a specific site.
	ListSiteClients(ctx context.Context, siteID SiteId, params *ListSiteClientsParams) (*ClientsResponse, error)

	// GetClientByID retrieves detailed information about a specific client.
	GetClientByID(ctx context.Context, siteID SiteId, clientID ClientId) (*NetworkClient, error)

	// Hotspot vouchers operations

	// ListHotspotVouchers retrieves a list of all hotspot vouchers for a specific site.
	ListHotspotVouchers(ctx context.Context, siteID SiteId, params *ListHotspotVouchersParams) (*HotspotVouchersResponse, error)

	// CreateHotspotVouchers creates one or more hotspot vouchers for temporary guest access.
	CreateHotspotVouchers(ctx context.Context, siteID SiteId, request *CreateVouchersRequest) (*HotspotVouchersResponse, error)

	// GetHotspotVoucher retrieves detailed information about a specific hotspot voucher.
	GetHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) (*HotspotVoucher, error)

	// DeleteHotspotVoucher permanently deletes a hotspot voucher.
	DeleteHotspotVoucher(ctx context.Context, siteID SiteId, voucherID openapi_types.UUID) error

	// DNS records operations

	// ListDNSRecords lists all static DNS records for a site.
	ListDNSRecords(ctx context.Context, site Site) ([]DNSRecord, error)

	// CreateDNSRecord creates a new static DNS record.
	CreateDNSRecord(ctx context.Context, site Site, record *DNSRecordInput) (*DNSRecord, error)

	// UpdateDNSRecord updates an existing DNS record.
	UpdateDNSRecord(ctx context.Context, site Site, recordID RecordId, record *DNSRecordInput) (*DNSRecord, error)

	// DeleteDNSRecord deletes a DNS record.
	DeleteDNSRecord(ctx context.Context, site Site, recordID RecordId) error

	// Firewall policies operations

	// ListFirewallPolicies lists all firewall policies for a site.
	ListFirewallPolicies(ctx context.Context, site Site) ([]FirewallPolicy, error)

	// CreateFirewallPolicy creates a new firewall policy.
	CreateFirewallPolicy(ctx context.Context, site Site, policy *FirewallPolicyInput) (*FirewallPolicy, error)

	// UpdateFirewallPolicy updates an existing firewall policy.
	UpdateFirewallPolicy(ctx context.Context, site Site, policyID PolicyId, policy *FirewallPolicyInput) (*FirewallPolicy, error)

	// DeleteFirewallPolicy permanently deletes a firewall policy.
	DeleteFirewallPolicy(ctx context.Context, site Site, policyID PolicyId) error

	// Traffic rules operations

	// ListTrafficRules lists all traffic rules for a site.
	ListTrafficRules(ctx context.Context, site Site) ([]TrafficRule, error)

	// CreateTrafficRule creates a new traffic rule.
	CreateTrafficRule(ctx context.Context, site Site, rule *TrafficRuleInput) (*TrafficRule, error)

	// UpdateTrafficRule updates an existing traffic rule.
	UpdateTrafficRule(ctx context.Context, site Site, ruleID RuleId, rule *TrafficRuleInput) (*TrafficRule, error)

	// DeleteTrafficRule permanently deletes a traffic rule.
	DeleteTrafficRule(ctx context.Context, site Site, ruleID RuleId) error

	// Dashboard operations

	// GetAggregatedDashboard retrieves aggregated dashboard statistics.
	GetAggregatedDashboard(ctx context.Context, site Site, params *GetAggregatedDashboardParams) (*AggregatedDashboard, error)

	// Topology operations (v2 API)

	// GetTopology retrieves the network topology graph for a site.
	GetTopology(ctx context.Context, site Site) (*Topology, error)

	// Active clients operations (v2 API)

	// ListActiveClients retrieves all currently connected clients with detailed connection information.
	ListActiveClients(ctx context.Context, site Site) ([]ActiveClient, error)

	// All devices operations (v2 API)

	// ListAllDevices retrieves all devices across all UniFi applications for a site.
	ListAllDevices(ctx context.Context, site Site) (*AllDevicesResponse, error)

	// Client details operations (v2 API)

	// GetActiveClientByMac retrieves detailed information about a specific active client by MAC address.
	GetActiveClientByMac(ctx context.Context, site Site, clientMac ClientMac) (*ActiveClientDetails, error)

	// WiFi statistics operations (v2 API)

	// GetWiFiStatsAPs retrieves WiFi statistics for all access points within a time range.
	GetWiFiStatsAPs(ctx context.Context, site Site, params *GetWiFiStatsAPsParams) (*WiFiStatsAPsResponse, error)

	// System logs operations (v2 API)

	// GetSystemLogs retrieves paginated system logs.
	GetSystemLogs(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// Firewall zones operations (v2 API)

	// ListFirewallZones retrieves all firewall zones for a site.
	ListFirewallZones(ctx context.Context, site Site) ([]FirewallZone, error)

	// GetFirewallZoneMatrix retrieves the firewall zone policy matrix for a site.
	GetFirewallZoneMatrix(ctx context.Context, site Site) ([]FirewallZoneMatrixEntry, error)

	// ISP status operations (v2 API)

	// GetISPStatus retrieves comprehensive ISP status with metrics and history.
	GetISPStatus(ctx context.Context, site Site) (*ISPStatus, error)

	// Client history operations (v2 API)

	// ListClientHistory retrieves historical client connection information.
	ListClientHistory(ctx context.Context, site Site) ([]ClientHistoryEntry, error)

	// Speedtest operations (v2 API)

	// ListSpeedtestHistory retrieves historical speedtest results.
	ListSpeedtestHistory(ctx context.Context, site Site) (*SpeedtestHistoryResponse, error)

	// GetSpeedtestLatestPerWan retrieves the latest speedtest per WAN interface.
	GetSpeedtestLatestPerWan(ctx context.Context, site Site) (*SpeedtestLatestPerWanResponse, error)

	// VPN operations (v2 API)

	// ListVPNConnections retrieves all VPN connections.
	ListVPNConnections(ctx context.Context, site Site) ([]VPNConnection, error)

	// Traffic routes operations (v2 API)

	// ListTrafficRoutes retrieves all traffic routes.
	ListTrafficRoutes(ctx context.Context, site Site) ([]TrafficRoute, error)

	// DHCP leases operations (v2 API)

	// ListActiveDHCPLeases retrieves all active DHCP leases with device fingerprint information.
	ListActiveDHCPLeases(ctx context.Context, site Site) (*DHCPLeasesResponse, error)

	// ISP health operations (v2 API)

	// GetISPHealth retrieves ISP health status with historical stats.
	GetISPHealth(ctx context.Context, site Site) (*ISPHealth, error)

	// GetISPHealthCompact retrieves compact ISP health status.
	GetISPHealthCompact(ctx context.Context, site Site) (*ISPHealthCompact, error)

	// NAT rules operations (v2 API)

	// ListNATRules retrieves all NAT (port forwarding) rules.
	ListNATRules(ctx context.Context, site Site) ([]NATRule, error)

	// Alerts and warnings operations (v2 API)

	// ListAlerts retrieves all alerts for the site.
	ListAlerts(ctx context.Context, site Site) (*AlertsResponse, error)

	// ListWarnings retrieves security warnings for the site.
	ListWarnings(ctx context.Context, site Site) (*WarningsResponse, error)

	// ListIPSAlerts retrieves intrusion prevention system alerts.
	ListIPSAlerts(ctx context.Context, site Site) (*IPSAlertsResponse, error)

	// Device groups operations (v2 API)

	// ListAPGroups retrieves all access point groups.
	ListAPGroups(ctx context.Context, site Site) ([]APGroup, error)

	// RADIUS operations (v2 API)

	// ListRADIUSProfiles retrieves all RADIUS authentication profiles.
	ListRADIUSProfiles(ctx context.Context, site Site) ([]RADIUSProfile, error)

	// ListRADIUSUsers retrieves RADIUS users.
	ListRADIUSUsers(ctx context.Context, site Site) ([]RADIUSUser, error)

	// Content filtering operations (v2 API)

	// ListContentFilteringRules retrieves content filtering rules for the site.
	ListContentFilteringRules(ctx context.Context, site Site) ([]ContentFilteringRule, error)

	// ListContentFilteringCategories retrieves all available content filtering categories.
	ListContentFilteringCategories(ctx context.Context, site Site) ([]string, error)

	// WiFi operations (v2 API)

	// GetWiFiConnectivity retrieves WiFi connection attempts and latency statistics.
	GetWiFiConnectivity(ctx context.Context, site Site) (*WiFiConnectivity, error)

	// GetWLANCapabilities retrieves WLAN capabilities like 6GHz and WPA3 support.
	GetWLANCapabilities(ctx context.Context, site Site) (*WLANCapabilities, error)

	// Features operations (v2 API)

	// ListFeatures retrieves list of features supported by the controller.
	ListFeatures(ctx context.Context, site Site) ([]string, error)

	// Teleport operations (v2 API)

	// ListTeleportInvitationHistory retrieves teleport invitation history.
	ListTeleportInvitationHistory(ctx context.Context, site Site) (*TeleportInvitationHistoryResponse, error)

	// Notifications operations (v2 API)

	// ListNotifications retrieves notifications for the site.
	ListNotifications(ctx context.Context, site Site) ([]Notification, error)

	// ACL operations (v2 API)

	// ListACLRules retrieves access control list rules.
	ListACLRules(ctx context.Context, site Site) ([]ACLRule, error)

	// Hotspot operations (v2 API)

	// GetHotspotInfo retrieves hotspot configuration status.
	GetHotspotInfo(ctx context.Context, site Site) (*HotspotInfo, error)

	// Client traffic operations (v2 API)

	// GetClientsTrafficControl retrieves traffic rule counts applied to clients.
	GetClientsTrafficControl(ctx context.Context, site Site) (*ClientsTrafficControl, error)

	// WireGuard operations (v2 API)

	// ListWireGuardUsers retrieves all WireGuard VPN users.
	ListWireGuardUsers(ctx context.Context, site Site) ([]WireGuardUser, error)

	// GetWireGuardExistingSubnets retrieves subnets already in use by WireGuard.
	GetWireGuardExistingSubnets(ctx context.Context, site Site) (*WireGuardSubnets, error)

	// ListVPNClientConnections retrieves all VPN client connections.
	ListVPNClientConnections(ctx context.Context, site Site) ([]VPNConnection, error)

	// BGP operations (v2 API)

	// GetBGPConfig retrieves BGP routing configuration.
	GetBGPConfig(ctx context.Context, site Site) ([]BGPConfig, error)

	// GetAllBGPConfig retrieves all BGP configurations across all devices.
	GetAllBGPConfig(ctx context.Context, site Site) ([]BGPConfig, error)

	// OSPF operations (v2 API)

	// ListOSPFRouters retrieves OSPF router configurations.
	ListOSPFRouters(ctx context.Context, site Site) ([]OSPFRouter, error)

	// ListOSPFNeighbors retrieves OSPF neighbor relationships.
	ListOSPFNeighbors(ctx context.Context, site Site) ([]OSPFNeighbor, error)

	// QoS operations (v2 API)

	// ListQoSRules retrieves Quality of Service rules.
	ListQoSRules(ctx context.Context, site Site) ([]QoSRule, error)

	// WAN SLA operations (v2 API)

	// ListWANSLAs retrieves WAN Service Level Agreement configurations.
	ListWANSLAs(ctx context.Context, site Site) ([]WANSLA, error)

	// Feature discovery operations (v2 API)

	// ListDescribedFeatures retrieves detailed list of features with availability and limitations.
	ListDescribedFeatures(ctx context.Context, site Site) ([]DescribedFeature, error)

	// ListVendorIDs retrieves list of known UniFi device vendor MAC address prefixes.
	ListVendorIDs(ctx context.Context, site Site) ([]string, error)

	// Gateway engine operations (v2 API)

	// GetGatewayEngineFeatures retrieves status of gateway engine features and their utilization.
	GetGatewayEngineFeatures(ctx context.Context, site Site) ([]GatewayEngineFeature, error)

	// GetGatewayEngineMostActiveNetworks retrieves list of most active networks by client count.
	GetGatewayEngineMostActiveNetworks(ctx context.Context, site Site) ([]MostActiveNetwork, error)

	// GetGatewayEngineUtilization retrieves CPU and memory utilization of the gateway.
	GetGatewayEngineUtilization(ctx context.Context, site Site) (*GatewayEngineUtilization, error)

	// Hotspot client operations (v2 API)

	// ListHotspotClients retrieves all clients connected via hotspot or regular network.
	ListHotspotClients(ctx context.Context, site Site) ([]HotspotClient, error)

	// Network diagnostics (v2 API)

	// GetLoopDetectionInfo retrieves information about detected network loops.
	GetLoopDetectionInfo(ctx context.Context, site Site) (*LoopDetectionInfo, error)

	// Port profile operations (v2 API)

	// GetPortProfileDefaults retrieves default port profile configuration.
	GetPortProfileDefaults(ctx context.Context, site Site) (*PortProfileDefaults, error)

	// SSL inspection operations (v2 API)

	// ListSSLInspectionCategories retrieves available SSL inspection content categories.
	ListSSLInspectionCategories(ctx context.Context, site Site) ([]SSLInspectionCategory, error)

	// ListSSLInspectionProfiles retrieves SSL inspection profile configurations.
	ListSSLInspectionProfiles(ctx context.Context, site Site) ([]SSLInspectionProfile, error)

	// Network configuration operations (v2 API)

	// GetLANEnrichedConfiguration retrieves enriched LAN network configurations.
	GetLANEnrichedConfiguration(ctx context.Context, site Site) ([]LANEnrichedConfiguration, error)

	// GetWANEnrichedConfiguration retrieves enriched WAN network configurations.
	GetWANEnrichedConfiguration(ctx context.Context, site Site) ([]WANEnrichedConfiguration, error)

	// Network score operations (v2 API)

	// GetNetworkScore retrieves overall network health score and subscores.
	GetNetworkScore(ctx context.Context, site Site) (*NetworkScore, error)

	// Static DNS operations (v2 API)

	// ListStaticDNSDevices retrieves device mappings for static DNS.
	ListStaticDNSDevices(ctx context.Context, site Site) ([]StaticDNSDevice, error)

	// Teleport operations (v2 API)

	// GetTeleportTokens retrieves Teleport VPN tokens.
	GetTeleportTokens(ctx context.Context, site Site) (*TeleportTokenResponse, error)

	// WiFiMan operations (v2 API)

	// GetWiFiManData retrieves WiFiMan diagnostic data for connected clients.
	GetWiFiManData(ctx context.Context, site Site) ([]WiFiManEntry, error)

	// Firewall defaults operations (v2 API)

	// GetFirewallZoneDefaults retrieves default firewall zone configurations.
	GetFirewallZoneDefaults(ctx context.Context, site Site) ([]FirewallZoneDefault, error)

	// GetFirewallPolicyDefaults retrieves default firewall policy configuration.
	GetFirewallPolicyDefaults(ctx context.Context, site Site) (*FirewallPolicyDefaults, error)

	// DNS over HTTPS operations (v2 API)

	// GetDOHDefaults retrieves DNS over HTTPS default settings.
	GetDOHDefaults(ctx context.Context, site Site) (*DOHDefaults, error)

	// GetDOHAvailableServerNames retrieves list of available DNS over HTTPS server names.
	GetDOHAvailableServerNames(ctx context.Context, site Site) (*DOHAvailableServers, error)

	// Network defaults operations (v2 API)

	// GetLANDefaults retrieves default LAN network configuration.
	GetLANDefaults(ctx context.Context, site Site) (*LANDefaults, error)

	// GetWANDefaults retrieves default WAN network configuration.
	GetWANDefaults(ctx context.Context, site Site) (*WANDefaults, error)

	// WLAN operations (v2 API)

	// GetWLANDefaults retrieves default WLAN configuration.
	GetWLANDefaults(ctx context.Context, site Site) ([]WLANDefaults, error)

	// GetWLANEnrichedConfiguration retrieves enriched WLAN configurations.
	GetWLANEnrichedConfiguration(ctx context.Context, site Site) ([]WLANEnrichedConfiguration, error)

	// VPN defaults operations (v2 API)

	// GetL2TPVPNDefaults retrieves default L2TP VPN configuration.
	GetL2TPVPNDefaults(ctx context.Context, site Site) (*L2TPVPNDefaults, error)

	// Global configuration operations (v2 API)

	// GetGlobalNetworkConfig retrieves global network configuration settings.
	GetGlobalNetworkConfig(ctx context.Context, site Site) (*GlobalNetworkConfig, error)

	// mDNS operations (v2 API)

	// ListMDNSServices retrieves available mDNS service definitions.
	ListMDNSServices(ctx context.Context, site Site) ([]MDNSService, error)

	// Combined firewall rules operations (v2 API)

	// ListCombinedFirewallRules retrieves combined traffic and firewall rules.
	ListCombinedFirewallRules(ctx context.Context, site Site) ([]CombinedFirewallRule, error)

	// GetFirewallRulesDefaults retrieves default firewall rule settings.
	GetFirewallRulesDefaults(ctx context.Context, site Site) ([]FirewallRulesDefault, error)

	// IP exclusion operations (v2 API)

	// GetExcludedIPs retrieves IPs excluded from device identification.
	GetExcludedIPs(ctx context.Context, site Site) (*ExcludedIPs, error)

	// Network suggestion operations (v2 API)

	// GetNetworkSuggestions retrieves suggested IP subnets for new networks.
	GetNetworkSuggestions(ctx context.Context, site Site) ([]NetworkSuggestion, error)

	// GetNetworkPortSuggestions retrieves available ports for new services.
	GetNetworkPortSuggestions(ctx context.Context, site Site) (*NetworkPortSuggestions, error)

	// WAN load balancing operations (v2 API)

	// GetWANLoadBalancingConfiguration retrieves WAN load balancing configuration.
	GetWANLoadBalancingConfiguration(ctx context.Context, site Site) (*WANLoadBalancingConfig, error)

	// GetWANLoadBalancingStatus retrieves WAN load balancing interface status.
	GetWANLoadBalancingStatus(ctx context.Context, site Site) (*WANLoadBalancingStatus, error)

	// GetWANNetworkGroups retrieves WAN network group configurations.
	GetWANNetworkGroups(ctx context.Context, site Site) (*WANNetworkGroups, error)

	// DDNS operations (v2 API)

	// GetDDNSProviders retrieves available Dynamic DNS providers.
	GetDDNSProviders(ctx context.Context, site Site) (*DDNSProviders, error)

	// Network status operations (v2 API)

	// GetNetworkStatus retrieves overall network health status.
	GetNetworkStatus(ctx context.Context, site Site) (*NetworkStatus, error)

	// Teleport operations (v2 API)

	// GetTeleportInvitationHistory retrieves Teleport VPN invitation history.
	GetTeleportInvitationHistory(ctx context.Context, site Site) (*TeleportInvitationHistory, error)

	// ULP operations (v2 API)

	// GetULPUsersGroups retrieves UniFi Local Portal users and groups.
	GetULPUsersGroups(ctx context.Context, site Site) (*ULPUsersGroups, error)

	// WAN Magic operations (v2 API)

	// GetWANMagicConfiguration retrieves WAN Magic (Teleport WAN) configuration.
	GetWANMagicConfiguration(ctx context.Context, site Site) (*WANMagicConfiguration, error)

	// System log operations (v2 API)

	// GetSystemLogSettings retrieves alert event settings for system logging.
	GetSystemLogSettings(ctx context.Context, site Site) (*SystemLogSettings, error)

	// GetSystemLogSettingsDefaults retrieves default alert event settings.
	GetSystemLogSettingsDefaults(ctx context.Context, site Site) (*SystemLogSettings, error)

	// Settings operations (v2 API)

	// GetSettingsConnectivityDefaults retrieves default connectivity settings.
	GetSettingsConnectivityDefaults(ctx context.Context, site Site) (*ConnectivityDefaults, error)

	// GetSettingsElementAdoptDefaults retrieves default element adoption settings.
	GetSettingsElementAdoptDefaults(ctx context.Context, site Site) (*ElementAdoptDefaults, error)

	// GetSettingsGlobalNATDefaults retrieves default global NAT settings.
	GetSettingsGlobalNATDefaults(ctx context.Context, site Site) (*GlobalNATDefaults, error)

	// GetSettingsGlobalSwitchDefaults retrieves default global switch settings.
	GetSettingsGlobalSwitchDefaults(ctx context.Context, site Site) (*GlobalSwitchDefaults, error)

	// GetSettingsIPSAdvancedFilteringAutoValues retrieves auto-computed IPS advanced filtering settings.
	GetSettingsIPSAdvancedFilteringAutoValues(ctx context.Context, site Site) (*IPSAdvancedFilteringSettings, error)

	// GetSettingsIPSAdvancedFilteringDefaults retrieves default IPS advanced filtering settings.
	GetSettingsIPSAdvancedFilteringDefaults(ctx context.Context, site Site) (*IPSAdvancedFilteringSettings, error)

	// GetSettingsIPSAvailableCategories retrieves available IPS categories.
	GetSettingsIPSAvailableCategories(ctx context.Context, site Site) ([]IPSCategory, error)

	// GetSettingsNetflowDefaults retrieves default netflow settings.
	GetSettingsNetflowDefaults(ctx context.Context, site Site) (*NetflowDefaults, error)

	// GetSettingsNTPDefaults retrieves default NTP settings.
	GetSettingsNTPDefaults(ctx context.Context, site Site) (*NTPDefaults, error)

	// GetSettingsRoamingAssistantDefaults retrieves default roaming assistant settings.
	GetSettingsRoamingAssistantDefaults(ctx context.Context, site Site) (*RoamingAssistantDefaults, error)

	// GetSettingsShortcuts retrieves configured shortcuts.
	GetSettingsShortcuts(ctx context.Context, site Site) ([]SettingsShortcut, error)

	// GetSettingsTeleportDefaults retrieves default teleport settings.
	GetSettingsTeleportDefaults(ctx context.Context, site Site) ([]TeleportSettingsDefaults, error)

	// GetSettingsTrafficFlowDefaults retrieves default traffic flow settings.
	GetSettingsTrafficFlowDefaults(ctx context.Context, site Site) (*TrafficFlowDefaults, error)

	// GetSettingsUSGDefaults retrieves default USG settings.
	GetSettingsUSGDefaults(ctx context.Context, site Site) (*USGDefaults, error)

	// GetSettingsWiFiAIDefaults retrieves default WiFi AI settings.
	GetSettingsWiFiAIDefaults(ctx context.Context, site Site) ([]WiFiAIDefaults, error)

	// SSL Inspection operations (v2 API)

	// GetSSLInspectionCertificates retrieves SSL inspection certificates.
	GetSSLInspectionCertificates(ctx context.Context, site Site) ([]SSLInspectionCertificate, error)

	// GetSSLInspectionCertificateActive retrieves the active SSL inspection certificate.
	GetSSLInspectionCertificateActive(ctx context.Context, site Site) (*SSLInspectionCertificate, error)

	// GetSSLInspectionFileExtensions retrieves file extensions for SSL inspection.
	GetSSLInspectionFileExtensions(ctx context.Context, site Site) ([]string, error)

	// GetSSLInspectionSearchEngines retrieves search engines for SSL inspection.
	GetSSLInspectionSearchEngines(ctx context.Context, site Site) ([]SSLInspectionSearchEngine, error)

	// GetSSLInspectionSetting retrieves SSL inspection setting.
	GetSSLInspectionSetting(ctx context.Context, site Site) (*SSLInspectionSetting, error)

	// GetSSLInspectionSettingDefaults retrieves default SSL inspection settings.
	GetSSLInspectionSettingDefaults(ctx context.Context, site Site) (*SSLInspectionSetting, error)

	// GetSSLInspectionProfilesDefaults retrieves default SSL inspection profile configuration.
	GetSSLInspectionProfilesDefaults(ctx context.Context, site Site) (*SSLInspectionProfileDefaults, error)

	// Miscellaneous v2 API operations

	// GetShadowModeInfo retrieves shadow mode information.
	GetShadowModeInfo(ctx context.Context, site Site) (*ShadowModeInfo, error)

	// GetStacking retrieves switch stacking configuration.
	GetStacking(ctx context.Context, site Site) ([]StackingConfig, error)

	// GetMCLAGGroups retrieves MC-LAG group configurations.
	GetMCLAGGroups(ctx context.Context, site Site) ([]MCLAGGroup, error)

	// GetDeviceWirelessLinks retrieves wireless mesh link configurations.
	GetDeviceWirelessLinks(ctx context.Context, site Site) ([]DeviceWirelessLink, error)

	// GetWireGuardUsersExistingSubnets retrieves WireGuard existing subnets.
	GetWireGuardUsersExistingSubnets(ctx context.Context, site Site) (*WireGuardExistingSubnets, error)

	// GetMagicSiteToSiteVPNConfigs retrieves Magic site-to-site VPN configurations.
	GetMagicSiteToSiteVPNConfigs(ctx context.Context, site Site) ([]MagicSiteToSiteVPNConfig, error)

	// GetObjectOrientedNetworkConfigs retrieves object-oriented network configurations.
	GetObjectOrientedNetworkConfigs(ctx context.Context, site Site) ([]ObjectOrientedNetworkConfig, error)

	// GetUtilizationLastDays retrieves utilization statistics for last days.
	GetUtilizationLastDays(ctx context.Context, site Site) (*UtilizationLastDays, error)

	// GetVPNClientConnections retrieves current VPN client connections.
	GetVPNClientConnections(ctx context.Context, site Site) (*VPNClientConnections, error)

	// System log display options operations (v2 API)

	// GetSystemLogRemoteSettings retrieves remote syslog settings.
	GetSystemLogRemoteSettings(ctx context.Context, site Site) (*SystemLogRemoteSettings, error)

	// GetSystemLogDisplayOptionsAdmins retrieves admin users for log filtering.
	GetSystemLogDisplayOptionsAdmins(ctx context.Context, site Site) ([]SystemLogAdmin, error)

	// GetSystemLogThreatDisplayOptionsClients retrieves clients for threat log filtering.
	GetSystemLogThreatDisplayOptionsClients(ctx context.Context, site Site) ([]SystemLogClient, error)

	// GetSystemLogTriggersDisplayOptionsHosts retrieves hosts for trigger log filtering.
	GetSystemLogTriggersDisplayOptionsHosts(ctx context.Context, site Site) ([]SystemLogHost, error)

	// GetSystemLogAPLogsDisplayOptionsAPs retrieves AP MAC addresses for AP log filtering.
	GetSystemLogAPLogsDisplayOptionsAPs(ctx context.Context, site Site) ([]string, error)

	// WiFi statistics operations with parameters (v2 API)

	// GetWiFiStatsRadios retrieves WiFi statistics per radio.
	GetWiFiStatsRadios(ctx context.Context, site Site, params *GetWiFiStatsRadiosParams) (*WiFiStatsRadiosResponse, error)

	// GetWiFiStatsChannelization retrieves WiFi channelization statistics over time.
	GetWiFiStatsChannelization(ctx context.Context, site Site, params *GetWiFiStatsChannelizationParams) (*WiFiStatsChannelizationResponse, error)

	// GetWiFiStatsDetails retrieves detailed WiFi statistics per connected client.
	GetWiFiStatsDetails(ctx context.Context, site Site, params *GetWiFiStatsDetailsParams) (*WiFiStatsDetailsResponse, error)

	// Traffic statistics operations (v2 API)

	// GetTrafficStats retrieves traffic statistics by client and application.
	GetTrafficStats(ctx context.Context, site Site, params *GetTrafficStatsParams) (*TrafficStatsResponse, error)

	// GetTrafficRate retrieves traffic rate time series data.
	GetTrafficRate(ctx context.Context, site Site, params *GetTrafficRateParams) ([]TrafficRateEntry, error)

	// GetUtilizationTimeRange retrieves system and ISP utilization over time range.
	GetUtilizationTimeRange(ctx context.Context, site Site, params *GetUtilizationTimeRangeParams) (*UtilizationTimeRangeResponse, error)

	// System log category operations (v2 API)

	// GetSystemLogCount retrieves system log counts by category.
	GetSystemLogCount(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogCountResponse, error)

	// GetSystemLogCritical retrieves critical system logs.
	GetSystemLogCritical(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogThreats retrieves threat detection logs.
	GetSystemLogThreats(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogAdminAccess retrieves admin access logs.
	GetSystemLogAdminAccess(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogAdminActivity retrieves admin activity logs.
	GetSystemLogAdminActivity(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogClientAlert retrieves client alert logs.
	GetSystemLogClientAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogDeviceAlert retrieves device alert logs.
	GetSystemLogDeviceAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogThreatAlert retrieves security threat alert logs.
	GetSystemLogThreatAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogUpdateAlert retrieves update alert logs.
	GetSystemLogUpdateAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogVPNAlert retrieves VPN alert logs.
	GetSystemLogVPNAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogNextAIAlert retrieves AI-powered threat detection alert logs.
	GetSystemLogNextAIAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// GetSystemLogSystemCriticalAlert retrieves system critical alert logs.
	GetSystemLogSystemCriticalAlert(ctx context.Context, site Site, request *SystemLogRequest) (*SystemLogResponse, error)

	// Firewall traffic flows operations (v2 API)

	// GetTrafficFlows retrieves firewall traffic flow data.
	GetTrafficFlows(ctx context.Context, site Site, request *TrafficFlowsRequest) (*TrafficFlowsResponse, error)

	// OpenVPN operations (v2 API)

	// GetOpenVPNCertificates retrieves OpenVPN certificates.
	GetOpenVPNCertificates(ctx context.Context, site Site) (*OpenVPNCertificatesResponse, error)

	// Alias operations (v2 API)

	// GetAliases retrieves client aliases for specified MAC addresses.
	GetAliases(ctx context.Context, site Site, request *AliasRequest) (*AliasResponse, error)

	// Clients metadata operations (v2 API)

	// GetClientsMetadata retrieves metadata for clients by their MAC addresses.
	GetClientsMetadata(ctx context.Context, site Site, macs []string) (*ClientsMetadataResponse, error)

	// Port system logs operations (v2 API)

	// GetPortSystemLogs retrieves system logs for a specific port.
	GetPortSystemLogs(ctx context.Context, site Site, request *PortSystemLogsRequest) (*PortSystemLogsResponse, error)

	// Application info operations (Integration API)

	// GetApplicationInfo retrieves general information about the UniFi Network application.
	GetApplicationInfo(ctx context.Context) (*ApplicationInfo, error)

	// Device statistics operations (Integration API)

	// GetDeviceLatestStatistics retrieves the latest real-time statistics of a device.
	GetDeviceLatestStatistics(ctx context.Context, siteID openapi_types.UUID, deviceID openapi_types.UUID) (*DeviceStatistics, error)

	// Device actions operations (Integration API)

	// ExecuteDeviceAction performs an action on a specific adopted device.
	ExecuteDeviceAction(ctx context.Context, siteID openapi_types.UUID, deviceID openapi_types.UUID, request DeviceActionRequest) error

	// RestartDevice restarts a specific adopted device.
	RestartDevice(ctx context.Context, siteID openapi_types.UUID, deviceID openapi_types.UUID) error

	// Port actions operations (Integration API)

	// ExecutePortAction performs an action on a specific device port.
	ExecutePortAction(ctx context.Context, siteID openapi_types.UUID, deviceID openapi_types.UUID, portIdx int32, request PortActionRequest) error

	// PowerCyclePort power cycles a PoE port (turns it off and on again).
	PowerCyclePort(ctx context.Context, siteID openapi_types.UUID, deviceID openapi_types.UUID, portIdx int32) error

	// Client actions operations (Integration API)

	// ExecuteClientAction performs an action on a specific connected client.
	ExecuteClientAction(ctx context.Context, siteID openapi_types.UUID, clientID openapi_types.UUID, request ClientActionRequest) (*ClientActionResponse, error)

	// AuthorizeGuestAccess authorizes a guest client for network access.
	AuthorizeGuestAccess(ctx context.Context, siteID openapi_types.UUID, clientID openapi_types.UUID, options *GuestAccessOptions) (*ClientActionResponse, error)

	// UnauthorizeGuestAccess revokes guest authorization and disconnects the client.
	UnauthorizeGuestAccess(ctx context.Context, siteID openapi_types.UUID, clientID openapi_types.UUID) (*ClientActionResponse, error)
}
