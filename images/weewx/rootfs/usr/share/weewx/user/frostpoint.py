import weewx
from weewx.engine import StdService

import weewx.units
weewx.units.obs_group_dict['frostpoint'] = 'group_temperature'

class AddFrostpoint(StdService):

    def __init__(self, engine, config_dict):
        # Initialize my superclass first:
        super(AddFrostpoint, self).__init__(engine, config_dict)
