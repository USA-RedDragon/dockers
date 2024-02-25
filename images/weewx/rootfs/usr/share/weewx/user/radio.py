import weewx
from weewx.engine import StdService

import weewx.units
weewx.units.obs_group_dict['inRSSI'] = 'group_db'
weewx.units.obs_group_dict['inSNR'] = 'group_db'
weewx.units.obs_group_dict['inNoise'] = 'group_db'
weewx.units.obs_group_dict['outRSSI'] = 'group_db'
weewx.units.obs_group_dict['outSNR'] = 'group_db'
weewx.units.obs_group_dict['outNoise'] = 'group_db'
weewx.units.obs_group_dict['tvoc'] = 'group_fraction'

class AddRadioUnits(StdService):

    def __init__(self, engine, config_dict):
        # Initialize my superclass first:
        super(AddRadioUnits, self).__init__(engine, config_dict)
