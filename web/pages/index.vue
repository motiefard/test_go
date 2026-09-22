<script setup>
const config = useRuntimeConfig()
const card = ref('')
const state = ref('idle')
const result = ref(null)
const errorMessage = ref('')

const persianErrorMessages = {
  VALIDATION_ERROR: 'اطلاعات کارت معتبر نیست.',
  BAD_REQUEST: 'پارامترهای ارسالی معتبر نیستند.',
  SERVER_ERROR: 'خطایی در سرور رخ داده است.',
  TIMEOUT: 'زمان پردازش درخواست به پایان رسیده است.',
  PROVIDER_ERROR: 'خطایی از سمت سرویس‌دهنده رخ داده است.',
  NOT_FOUND: 'اطلاعات موردنظر یافت نشد.',
  LOGIC_ERROR: 'خطایی در پردازش درخواست رخ داده است.',
  UNAUTHORIZED: 'خطای احراز هویت.',
  ACCESS_DENIED: 'دسترسی به سرویس‌دهنده رد شد.',
  UNAVAILABLE: 'سامانه ارائه‌دهنده در حال حاضر در دسترس نیست.',
  INTERNAL_ERROR: 'خطای داخلی رخ داده است.'
}

function getPersianErrorMessage(error) {
  return persianErrorMessages[error?.code] || 'درخواست ناموفق بود.'
}

function formatIban(iban) {
  return (iban || '').replace(/(.{4})/g, '$1 ').trim()
}

function formatCardInput(value) {
  const digits = value.replace(/\D/g, '').slice(0, 16)
  return digits.replace(/(\d{4})(?=\d)/g, '$1 ')
}

function onCardInput(event) {
  card.value = formatCardInput(event.target.value)
}

async function submit() {
  state.value = 'loading'
  result.value = null
  errorMessage.value = ''
  try {
    const payload = { card: card.value.replace(/\s/g, '') }
    const res = await fetch(`${config.public.apiBase}/api/v1/card-to-sheba`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    const data = await res.json()
    if (!res.ok) {
      state.value = data?.error?.code === 'VALIDATION_ERROR' ? 'validation' : 'service'
      errorMessage.value = getPersianErrorMessage(data?.error)
      return
    }
    result.value = data
    state.value = 'success'
  } catch {
    state.value = 'service'
    errorMessage.value = 'ارتباط با سرویس تبدیل امکان‌پذیر نیست.'
  }
}
</script>

<template>
  <main class="min-h-screen bg-slate-50 text-slate-900">
    <div class="mx-auto flex min-h-screen max-w-xl flex-col justify-center px-4 py-10">
      <section class="rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-200 sm:p-8">
        <h1 class="text-2xl font-semibold tracking-tight">Card to SHEBA</h1>
        <p class="mt-2 text-sm text-slate-600">
          Enter a 16-digit bank card number to convert it to a SHEBA (IBAN).
        </p>

        <form class="mt-6 space-y-4" @submit.prevent="submit">
          <label class="block text-sm font-medium" for="card">Card number</label>
          <input
            id="card"
            name="card"
            inputmode="numeric"
            autocomplete="cc-number"
            maxlength="19"
            required
            :disabled="state === 'loading'"
            class="w-full rounded-xl border border-slate-300 px-4 py-3 text-lg tracking-widest outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-100 disabled:bg-slate-100"
            placeholder="6037 9975 9993 9999"
            :value="card"
            @input="onCardInput"
          />

          <button
            type="submit"
            class="w-full rounded-xl bg-emerald-600 px-4 py-3 font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-emerald-300"
            :disabled="state === 'loading' || card.replace(/\s/g, '').length !== 16"
          >
            {{ state === 'loading' ? 'Converting…' : 'Convert to SHEBA' }}
          </button>
        </form>

        <p v-if="state === 'loading'" class="mt-4 text-sm text-slate-500">Calling conversion service…</p>
        <p v-else-if="state === 'validation'" dir="rtl" class="mt-4 rounded-lg bg-amber-50 px-3 py-2 text-right text-sm text-amber-800">{{ errorMessage }}</p>
        <p v-else-if="state === 'service'" dir="rtl" class="mt-4 rounded-lg bg-rose-50 px-3 py-2 text-right text-sm text-rose-800">{{ errorMessage }}</p>

        <div v-else-if="state === 'success' && result" class="mt-6 space-y-3 rounded-xl bg-emerald-50 p-4">
          <p class="text-xs font-medium uppercase tracking-wide text-emerald-800">SHEBA / IBAN</p>
          <p class="break-all font-mono text-xl font-semibold text-emerald-950">{{ formatIban(result.iban) }}</p>
          <dl class="grid grid-cols-1 gap-2 text-sm text-slate-700 sm:grid-cols-2">
            <div>
              <dt class="text-slate-500">Bank</dt>
              <dd>{{ result.bankName }} ({{ result.englishBankName }})</dd>
            </div>
            <div>
              <dt class="text-slate-500">Owner</dt>
              <dd>{{ result.depositOwners || '—' }}</dd>
            </div>
            <div>
              <dt class="text-slate-500">Account</dt>
              <dd class="font-mono">{{ result.deposit }}</dd>
            </div>
            <div>
              <dt class="text-slate-500">Status</dt>
              <dd>{{ result.depositDescription }}</dd>
            </div>
          </dl>
        </div>
      </section>
    </div>
  </main>
</template>
