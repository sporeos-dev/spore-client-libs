#include "witness.h"

namespace spore
{
    std::string_view Witness::getBody() const  { return body; }
    void Witness::setBody(std::string_view v)  { body = v;   }

}  // namespace spore
