#include "pulsego.h"

void card_info_cb(pa_context *c, const pa_card_info *i, int eol, void *userdata) {
  go_card_info_cb((pa_card_info*)i, eol, userdata);
}

void sink_info_cb(pa_context *c, const pa_sink_info *i, int eol, void *userdata) {
  go_sink_info_cb((pa_sink_info*)i, eol, userdata);
}

void context_subscribe_cb(pa_context *c, pa_subscription_event_type_t t, uint32_t idx, void *userdata) {
  go_context_subscribe_cb(t, idx, userdata);
}