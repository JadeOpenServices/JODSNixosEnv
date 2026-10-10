# Bezel Quick Keys (HP ZBook x2: six keys on the right edge). ODDC's
# oddc-quick-keys service sends neutral KEY_MACRO codes; the meaning is ours.
# Buttons count top to bottom; button3 is the preset switch and takes no key.
# A model without Quick Keys has no such service, so these overrides do nothing
# there.
{ ... }:
{
  oddc.overrides.policy.input.quickKeys.keymap = {
    # Desktop: on-screen keyboard, screenshot, lock, brightness.
    preset1 = {
      button1 = "KEY_KEYBOARD";
      button2 = "KEY_PRINT";
      button4 = "KEY_SCREENLOCK";
      button5 = "KEY_BRIGHTNESSUP";
      button6 = "KEY_BRIGHTNESSDOWN";
    };
    # Media.
    preset2 = {
      button1 = "KEY_PLAYPAUSE";
      button2 = "KEY_PREVIOUSSONG";
      button4 = "KEY_NEXTSONG";
      button5 = "KEY_MUTE";
      button6 = "KEY_MICMUTE";
    };
    # preset3 keeps ODDC's KEY_MACRO11-15, free for the user's own binds.
  };
}
