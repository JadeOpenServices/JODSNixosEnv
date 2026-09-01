{ lib, ... }:
{
  options.services.kmscon.config = lib.mkOption {
    type = lib.types.anything;
    default = null;
    description = "Compatibility option for deprecated services.kmscon.config";
  };
}
