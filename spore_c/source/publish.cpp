#include "publish.h"

namespace spore
{
    std::string_view Publish::getTopic() const { return topic; }
    void Publish::setTopic(std::string_view v) { topic = v;   }

}  // namespace spore
