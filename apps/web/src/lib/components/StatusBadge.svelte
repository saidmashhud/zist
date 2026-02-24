<script lang="ts">
  import type { BookingStatus } from '$lib/types';

  let { status }: { status: BookingStatus } = $props();

  const labels: Record<BookingStatus, string> = {
    pending_host_approval: 'Awaiting approval',
    payment_pending:       'Payment required',
    confirmed:             'Confirmed',
    cancelled_by_guest:    'Cancelled',
    cancelled_by_host:     'Cancelled by host',
    rejected:              'Declined',
    failed:                'Payment failed',
    completed:             'Completed',
  };

  const styles: Record<BookingStatus, string> = {
    pending_host_approval: 'bg-yellow-50 text-yellow-700 border border-yellow-200',
    payment_pending:       'bg-blue-50 text-blue-700 border border-blue-200',
    confirmed:             'bg-green-50 text-green-700 border border-green-200',
    cancelled_by_guest:    'bg-gray-100 text-gray-500 border border-gray-200',
    cancelled_by_host:     'bg-gray-100 text-gray-500 border border-gray-200',
    rejected:              'bg-red-50 text-red-600 border border-red-200',
    failed:                'bg-red-50 text-red-600 border border-red-200',
    completed:             'bg-gray-100 text-gray-600 border border-gray-200',
  };
</script>

<span class="shrink-0 rounded-full px-3 py-1 text-xs font-medium {styles[status]}">
  {labels[status]}
</span>
