export default {
  vendorHall: {
    title: 'Vendor Hall', description: 'Inspect real-traffic health for provider accounts and control their scheduling state.', live: 'Live Monitor data', window: 'Time window', search: 'Search account, platform, or group', allStatuses: 'All statuses', details: 'Expand account details', empty: 'No provider accounts match these filters', failed: 'Failed to load vendor data',
    summary: { total: 'Connected accounts', accounts: 'provider accounts', healthy: 'Healthy', running: 'Available for scheduling', paused: 'Paused', manual: 'Temporarily or permanently off', availability: 'Average availability' },
    columns: { account: 'Provider account', multiplier: 'Multiplier', latency: 'User latency', cache: 'Cache hit', availability: 'Availability', ttft: 'User TTFT trend', status: 'Scheduling' },
    metrics: { rateMultiplier: 'Upstream multiplier', userLatency: 'User latency', cache: 'Cache', availability: 'Availability', userTtft: 'User TTFT P95', requests: 'Requests', updated: 'Last collected', averageLatency: 'Average latency' },
    status: { schedulable: 'Scheduling', paused: 'Paused', disabled: 'Disabled', unknown: 'Unknown' },
    sort: { availability: 'By availability', cache: 'By cache hit', ttft: 'By TTFT', requests: 'By requests' },
    actions: { usage: 'View usage', manage: 'Manage account', pause: 'Pause 1 hour', disable: 'Disable scheduling' },
    confirm: { pauseTitle: 'Pause account scheduling', pauseMessage: 'This account will not receive new scheduled requests for one hour, then resume automatically.', disableTitle: 'Disable account scheduling', disableMessage: 'This account will remain out of scheduling until an administrator enables it. Credentials and history are preserved.' },
    success: { paused: 'Scheduling paused for one hour', disabled: 'Account scheduling disabled' },
  },
}
