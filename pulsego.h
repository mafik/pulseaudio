#ifndef _PULSEGO_H
#define _PULSEGO_H

#include <pulse/pulseaudio.h>

extern void go_card_info_cb(pa_card_info *i, int eol, void *userdata);
extern void go_sink_info_cb(pa_sink_info *i, int eol, void *userdata);
void go_context_subscribe_cb(pa_subscription_event_type_t t, uint32_t idx, void *userdata);

void card_info_cb(pa_context *c, const pa_card_info *i, int eol, void *userdata);
void sink_info_cb(pa_context *c, const pa_sink_info *i, int eol, void *userdata);
void context_subscribe_cb(pa_context *c, pa_subscription_event_type_t t, uint32_t idx, void *userdata);

#endif  // _PULSEGO_H