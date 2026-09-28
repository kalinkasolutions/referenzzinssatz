const form = document.getElementById("subscribe-form");
const mailInput = document.getElementById("mail");
const submitButton = form.querySelector("button");
const status = document.getElementById("form-status");

const messages = {
  confirmation_pending: "Fast geschafft! Bitte bestätigen Sie Ihre Anmeldung über den Link, den wir Ihnen soeben geschickt haben.",
  invalid_email: "Bitte geben Sie eine gültige E-Mail-Adresse ein.",
  captcha_failed: "Die Sicherheitsprüfung ist fehlgeschlagen. Bitte versuchen Sie es erneut.",
  captcha_unavailable: "Die Sicherheitsprüfung konnte nicht geladen werden. Ein Werbeblocker kann die Ursache sein.",
  mail_failed: "Die Bestätigungs-E-Mail konnte nicht gesendet werden. Bitte versuchen Sie es später erneut.",
  unknown: "Die Anmeldung hat nicht geklappt. Bitte versuchen Sie es später erneut.",
};

function showStatus(kind, text) {
  status.dataset.kind = kind;
  status.textContent = text;
}

function requestCaptchaToken(siteKey) {
  return new Promise((resolve, reject) => {
    const recaptcha = window.grecaptcha?.enterprise;
    if (!recaptcha) {
      reject(new Error("captcha_unavailable"));
      return;
    }
    recaptcha.ready(() => recaptcha.execute(siteKey, { action: "subscribe" }).then(resolve, reject));
  });
}

async function subscribe(mail) {
  let reCaptchaToken;
  try {
    reCaptchaToken = await requestCaptchaToken(form.dataset.siteKey);
  } catch {
    return "captcha_unavailable";
  }

  const response = await fetch("/subscribe", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mail, reCaptchaToken }),
  });
  const result = await response.json().catch(() => ({}));
  return response.ok ? "confirmation_pending" : result.error;
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  submitButton.disabled = true;
  showStatus("pending", "Wird gesendet …");

  try {
    const outcome = await subscribe(mailInput.value.trim());
    if (outcome === "confirmation_pending") {
      form.reset();
      showStatus("success", messages.confirmation_pending);
    } else {
      showStatus("error", messages[outcome] ?? messages.unknown);
    }
  } catch {
    showStatus("error", messages.unknown);
  } finally {
    submitButton.disabled = false;
  }
});
